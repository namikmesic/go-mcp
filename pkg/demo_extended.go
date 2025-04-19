package demo

import (
	"context"
	"io"
)

// EmptyInterface is an interface with no methods at all
type EmptyInterface interface{}

// InterfaceWithoutDocs has no documentation comments
type InterfaceWithoutDocs interface {
	DoSomething()
}

// EmptyMethodsInterface has methods without parameters or return values
type EmptyMethodsInterface interface {
	// DoNothing has documentation but no parameters or return values
	DoNothing()

	// NoParamsWithReturn has no parameters but has return values
	NoParamsWithReturn() error

	// ParamsNoReturn has parameters but no return values
	ParamsNoReturn(ctx context.Context)
}

// EmbeddingInterface embeds other interfaces
type EmbeddingInterface interface {
	// Embed standard library interfaces
	io.Reader
	io.Writer
	io.Closer

	// Embed your own interfaces
	EmptyInterface

	// Add a method of its own
	ExtraMethod() string
}

// ComplexTypesInterface demonstrates less common types
type ComplexTypesInterface interface {
	// Method with variadic parameter
	Variadic(first string, rest ...interface{})

	// Method with function parameter
	WithCallback(callback func(string) error) bool

	// Method with complex map and channel return types
	ComplexReturn() (map[string]<-chan []string, context.CancelFunc)

	// Method with named return values
	NamedReturns() (count int, err error)
}

// Actually implement an interface to test implementation detection
type ConcreteType struct{}

func (c ConcreteType) DoSomething() {
	// Implementation
}

// Implement with pointer receiver to test pointer implementation detection
type PointerImplementer struct{}

func (p *PointerImplementer) DoSomething() {
	// Implementation
}

// Multiple implementations of the same interface
type AnotherImplementation struct{}

func (a AnotherImplementation) DoSomething() {
	// Different implementation
}

// Type that implements multiple interfaces
type MultiInterfaceImplementer struct{}

func (m MultiInterfaceImplementer) DoSomething() {
	// Implementation for InterfaceWithoutDocs
}

func (m MultiInterfaceImplementer) DoNothing() {
	// Implementation for EmptyMethodsInterface
}

func (m MultiInterfaceImplementer) NoParamsWithReturn() error {
	return nil
}

func (m MultiInterfaceImplementer) ParamsNoReturn(ctx context.Context) {
	// Implementation
}

// GenericInterface demonstrates a generic interface
type GenericInterface[T any] interface {
	Process(data T) T
	Compare(a, b T) int
}

// EmbeddedInterface is used for embedding
type EmbeddedInterface interface {
	EmbeddedMethod() string
}
