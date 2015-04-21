package testsuite_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Let", func() {
	title := func() string { return "Dr." }
	firstName := func() string { return "Karl" }
	lastName := func() string { return "Marx" }
	fullName := func() string { return title() + " " + firstName() + " " + lastName() }
	storedVal := "A"
	counter := 0
	memoizedCounter := Let(func() interface{} { counter += 1; return counter })
	fullNameCalculatedInOuterBefore := ""

	BeforeEach(func() {
		storedVal = storedVal + "B"
		fullNameCalculatedInOuterBefore = fullName()
	})

	It("should be initially correct", func() {
		Expect(fullName()).To(Equal("Dr. Karl Marx"))
	})

	Context("in a nested context", func() {
		title = func() string { return "Ms." }
		storedVal = storedVal + "C"

		BeforeEach(func() {
			storedVal = storedVal + "D"
		})

		It("should be overridable", func() {
			Expect(fullName()).To(Equal("Ms. Karl Marx"))
		})

		It("should evaluate BeforeEaches with properly-generated bodies", func() {
			Expect(storedVal).To(Equal("ACBD"))
		})

		It("should work like lets even for way-outer befores", func() {
			Expect(fullNameCalculatedInOuterBefore).To(Equal("Ms. Karl Marx"))
		})
	})

	It("should still be correct after a context override", func() {
		Expect(fullName()).To(Equal("Dr. Karl Marx"))
	})

	It("should evaluate BeforeEaches with properly-generated bodies", func() {
		Expect(storedVal).To(Equal("AB"))
	})

	It("should memoize when Let() is used", func() {
		Expect(memoizedCounter().(int)).To(Equal(1))
		Expect(memoizedCounter().(int)).To(Equal(1))
		Expect(memoizedCounter().(int)).To(Equal(1))
		Expect(counter).To(Equal(1))

		counter = 2
		Expect(memoizedCounter().(int)).To(Equal(1))
	})
})
