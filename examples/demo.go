package demo

import (
	"context"
	"time"
)

// Animal represents a basic animal interface
type Animal interface {
	Speak() string
	Move() string
	Eat() string
}

// Vehicle defines transportation capabilities
type Vehicle interface {
	Start() error
	Stop() error
	GetSpeed() float64
	GetFuelLevel() float64
}

// Database operations interface
type Database interface {
	Connect() error
	Disconnect() error
	Query(query string) ([]interface{}, error)
	Insert(data interface{}) error
	Update(id string, data interface{}) error
	Delete(id string) error
}

// Logger interface for logging operations
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// PaymentProcessor handles payment operations
type PaymentProcessor interface {
	ProcessPayment(amount float64, currency string) error
	RefundPayment(paymentID string) error
	GetPaymentStatus(paymentID string) (string, error)
	ValidateCard(cardNumber string) bool
}

// ComplexData represents a complex data structure
type ComplexData struct {
	ID        string
	Data      []byte
	Metadata  map[string]interface{}
	Timestamp time.Time
}

// DataProcessor handles complex data processing
type DataProcessor interface {
	// Process handles the main data processing
	Process(ctx context.Context, data *ComplexData) (*ComplexData, error)

	// BatchProcess handles multiple items
	BatchProcess(ctx context.Context, items []*ComplexData) ([]*ComplexData, error)

	// Validate checks if the data is valid
	Validate(data *ComplexData) (bool, []string)

	// Transform converts data between formats
	Transform(data *ComplexData, format string) (*ComplexData, error)
}

// CacheManager handles caching operations
type CacheManager interface {
	// Get retrieves an item from cache
	Get(key string) (*ComplexData, bool)

	// Set stores an item in cache
	Set(key string, value *ComplexData, ttl time.Duration) error

	// Delete removes an item from cache
	Delete(key string) error

	// Clear removes all items from cache
	Clear() error

	// GetStats returns cache statistics
	GetStats() *CacheStats
}

// CacheStats represents cache statistics
type CacheStats struct {
	HitCount  int64
	MissCount int64
	Size      int64
}

// EventHandler manages event processing
type EventHandler interface {
	// HandleEvent processes a single event
	HandleEvent(event *Event) error

	// HandleEvents processes multiple events
	HandleEvents(events []*Event) []error

	// RegisterHandler registers a new event handler
	RegisterHandler(eventType string, handler func(*Event) error) error

	// UnregisterHandler removes an event handler
	UnregisterHandler(eventType string) error
}

// Event represents an event in the system
type Event struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
	Source    string
}

// ResourceManager handles resource allocation and cleanup
type ResourceManager interface {
	// Allocate allocates a new resource
	Allocate(ctx context.Context, spec *ResourceSpec) (*Resource, error)

	// Deallocate releases a resource
	Deallocate(resource *Resource) error

	// GetStatus returns the current status of a resource
	GetStatus(resource *Resource) (*ResourceStatus, error)

	// Watch monitors resource changes
	Watch(ctx context.Context, resource *Resource) (<-chan *ResourceStatus, error)
}

// ResourceSpec defines resource requirements
type ResourceSpec struct {
	Type        string
	Capacity    int64
	Properties  map[string]string
	Constraints []string
}

// Resource represents an allocated resource
type Resource struct {
	ID        string
	Spec      *ResourceSpec
	Status    string
	CreatedAt time.Time
}

// ResourceStatus represents the current status of a resource
type ResourceStatus struct {
	State     string
	Usage     float64
	UpdatedAt time.Time
	Error     error
}

// MetricsCollector handles metrics collection and reporting
type MetricsCollector interface {
	// RecordMetric records a single metric
	RecordMetric(name string, value float64, labels map[string]string) error

	// RecordMetrics records multiple metrics
	RecordMetrics(metrics []*Metric) error

	// GetMetrics retrieves metrics for a time range
	GetMetrics(start, end time.Time) ([]*Metric, error)

	// ExportMetrics exports metrics in a specific format
	ExportMetrics(format string) ([]byte, error)
}

// Metric represents a single metric measurement
type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}
