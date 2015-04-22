package suite

import (
	"math/rand"
	"time"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/internal/containernode"
	"github.com/onsi/ginkgo/internal/failer"
	"github.com/onsi/ginkgo/internal/leafnodes"
	"github.com/onsi/ginkgo/internal/spec"
	"github.com/onsi/ginkgo/internal/specrunner"
	"github.com/onsi/ginkgo/internal/writer"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/ginkgo/types"
)

type ginkgoTestingT interface {
	Fail()
}

type Suite struct {
	topLevelContainer *containernode.ContainerNode
	currentContainer  *containernode.ContainerNode
	containerIndex    int
	beforeSuiteNode   leafnodes.SuiteNode
	afterSuiteNode    leafnodes.SuiteNode
	runner            *specrunner.SpecRunner
	failer            *failer.Failer
	running           bool
	currentContainerCoords      []int
	currentIndex    int
	seekingContainerCoords *[]int
	seekingItIndex    int
	foundCollated     *containernode.CollatedNodes
}

func New(failer *failer.Failer) *Suite {
	topLevelContainer := containernode.New("[Top Level]", types.FlagTypeNone, types.CodeLocation{}, []int{0}, nil)

	return &Suite{
		topLevelContainer: topLevelContainer,
		currentContainer:  topLevelContainer,
		failer:            failer,
		containerIndex:    1,
		currentContainerCoords:      make([]int, 0, 10),
	}
}

func (suite *Suite) Run(t ginkgoTestingT, description string, reporters []reporters.Reporter, writer writer.WriterInterface, config config.GinkgoConfigType) (bool, bool) {
	if config.ParallelTotal < 1 {
		panic("ginkgo.parallel.total must be >= 1")
	}

	if config.ParallelNode > config.ParallelTotal || config.ParallelNode < 1 {
		panic("ginkgo.parallel.node is one-indexed and must be <= ginkgo.parallel.total")
	}

	r := rand.New(rand.NewSource(config.RandomSeed))
	suite.topLevelContainer.Shuffle(r)
	specs := suite.generateSpecs(description, config)
	suite.runner = specrunner.New(description, suite.beforeSuiteNode, specs, suite.afterSuiteNode, reporters, writer, config)

	suite.running = true
	success := suite.runner.Run()
	if !success {
		t.Fail()
	}
	return success, specs.HasProgrammaticFocus()
}

func (suite *Suite) generateSpecs(description string, config config.GinkgoConfigType) *spec.Specs {
	specsSlice := []*spec.Spec{}
	suite.topLevelContainer.BackPropagateProgrammaticFocus()
	for _, collatedNodes := range suite.topLevelContainer.Collate() {
		specsSlice = append(specsSlice, spec.New(collatedNodes.Subject, collatedNodes.Containers, config.EmitSpecProgress))
	}

	specs := spec.NewSpecs(specsSlice)

	if config.RandomizeAllSpecs {
		specs.Shuffle(rand.New(rand.NewSource(config.RandomSeed)))
	}

	specs.ApplyFocus(description, config.FocusString, config.SkipString)

	if config.SkipMeasurements {
		specs.SkipMeasurements()
	}

	if config.ParallelTotal > 1 {
		specs.TrimForParallelization(config.ParallelTotal, config.ParallelNode)
	}

	return specs
}

func (suite *Suite) CurrentRunningSpecSummary() (*types.SpecSummary, bool) {
	return suite.runner.CurrentSpecSummary()
}

func (suite *Suite) SetBeforeSuiteNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.beforeSuiteNode != nil {
		panic("You may only call BeforeSuite once!")
	}
	suite.beforeSuiteNode = leafnodes.NewBeforeSuiteNode(body, codeLocation, timeout, suite.failer)
}

func (suite *Suite) SetAfterSuiteNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.afterSuiteNode != nil {
		panic("You may only call AfterSuite once!")
	}
	suite.afterSuiteNode = leafnodes.NewAfterSuiteNode(body, codeLocation, timeout, suite.failer)
}

func (suite *Suite) SetSynchronizedBeforeSuiteNode(bodyA interface{}, bodyB interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.beforeSuiteNode != nil {
		panic("You may only call BeforeSuite once!")
	}
	suite.beforeSuiteNode = leafnodes.NewSynchronizedBeforeSuiteNode(bodyA, bodyB, codeLocation, timeout, suite.failer)
}

func (suite *Suite) SetSynchronizedAfterSuiteNode(bodyA interface{}, bodyB interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.afterSuiteNode != nil {
		panic("You may only call AfterSuite once!")
	}
	suite.afterSuiteNode = leafnodes.NewSynchronizedAfterSuiteNode(bodyA, bodyB, codeLocation, timeout, suite.failer)
}

func startsWith(data []int, prefix []int) bool {
	if len(data) < len(prefix) {
		return false
	}

	for i := range prefix {
		if data[i] != prefix[i] {
			return false
		}
	}

	return true
}

func equals(dataA []int, dataB []int) bool {
	if len(dataA) != len(dataB) {
		return false
	}

	return startsWith(dataA, dataB)
}

func (suite *Suite) PushContainerNode(text string, body func(), flag types.FlagType, codeLocation types.CodeLocation) {
	if suite.foundCollated != nil {
		return
	}

	rerunFunc := func(targetCoords []int, itIndex int) ([]*containernode.ContainerNode, leafnodes.SubjectNode) {
		suite.topLevelContainer = containernode.New("[Fake Top Leve]", types.FlagTypeNone, types.CodeLocation{}, []int{0}, nil)
		suite.currentContainer = suite.topLevelContainer
		suite.currentContainerCoords = make([]int, 1, 10)
		suite.currentContainerCoords[0] = targetCoords[0]
		suite.currentIndex = 0
		suite.seekingContainerCoords = &targetCoords
		suite.seekingItIndex = itIndex
		suite.running = false
		body()
		suite.running = true
		suite.seekingContainerCoords = nil
		fc := suite.foundCollated
		suite.foundCollated = nil
		return fc.Containers, fc.Subject
	}

	thisContainerCoords := make([]int, len(suite.currentContainerCoords) + 1)
	copy(thisContainerCoords, suite.currentContainerCoords)
	thisContainerCoords[len(suite.currentContainerCoords)] = suite.currentIndex

	if suite.seekingContainerCoords == nil || startsWith(*suite.seekingContainerCoords, thisContainerCoords) {
		priorItIndex := suite.currentIndex

		suite.currentContainerCoords = append(suite.currentContainerCoords, suite.currentIndex)
		suite.currentIndex = 0

		container := containernode.New(text, flag, codeLocation, thisContainerCoords, rerunFunc)
		suite.currentContainer.PushContainerNode(container)

		previousContainer := suite.currentContainer
		suite.currentContainer = container
		suite.containerIndex++

		body()

		suite.containerIndex--
		suite.currentContainer = previousContainer
		suite.currentIndex = priorItIndex
		suite.currentContainerCoords = suite.currentContainerCoords[:len(suite.currentContainerCoords) - 1]
	} else {
		if suite.seekingContainerCoords != nil {
		}
	}

	suite.currentIndex += 1
}

func (suite *Suite) PushItNode(text string, body interface{}, flag types.FlagType, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.foundCollated != nil {
		return
	}

	if suite.running {
		suite.failer.Fail("You may only call It from within a Describe or Context", codeLocation)
	}


	if suite.seekingContainerCoords != nil {
	}

	itMatches := suite.seekingContainerCoords == nil || equals(suite.currentContainerCoords, *suite.seekingContainerCoords) && suite.currentIndex == suite.seekingItIndex

	if itMatches {
		suite.currentContainer.PushSubjectNode(leafnodes.NewItNode(text, body, flag, codeLocation, timeout, suite.failer, suite.containerIndex, suite.currentIndex))

		if suite.seekingContainerCoords != nil {

			collatedNodes := suite.topLevelContainer.Collate()
			if len(collatedNodes) != 1 {
				panic("wha?!?!?!")
			}
			suite.foundCollated = &collatedNodes[0]
		}
	}

	suite.currentIndex += 1
}

func (suite *Suite) PushMeasureNode(text string, body interface{}, flag types.FlagType, codeLocation types.CodeLocation, samples int) {
	if suite.running {
		suite.failer.Fail("You may only call Measure from within a Describe or Context", codeLocation)
	}
	suite.currentContainer.PushSubjectNode(leafnodes.NewMeasureNode(text, body, flag, codeLocation, samples, suite.failer, suite.containerIndex))
}

func (suite *Suite) PushBeforeEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.running {
		suite.failer.Fail("You may only call BeforeEach from within a Describe or Context", codeLocation)
	}
	suite.currentContainer.PushSetupNode(leafnodes.NewBeforeEachNode(body, codeLocation, timeout, suite.failer, suite.containerIndex))
}

func (suite *Suite) PushJustBeforeEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.running {
		suite.failer.Fail("You may only call JustBeforeEach from within a Describe or Context", codeLocation)
	}
	suite.currentContainer.PushSetupNode(leafnodes.NewJustBeforeEachNode(body, codeLocation, timeout, suite.failer, suite.containerIndex))
}

func (suite *Suite) PushAfterEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.running {
		suite.failer.Fail("You may only call AfterEach from within a Describe or Context", codeLocation)
	}
	suite.currentContainer.PushSetupNode(leafnodes.NewAfterEachNode(body, codeLocation, timeout, suite.failer, suite.containerIndex))
}
