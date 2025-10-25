// Package internal provides internal implementation details for servicex.
package internal

import (
	"fmt"
	"reflect"
	"sync"
)

// Container is a simple dependency injection container.
type Container struct {
	mu           sync.RWMutex
	constructors map[reflect.Type]reflect.Value
	instances    map[reflect.Type]reflect.Value
	building     map[reflect.Type]bool
}

// NewContainer creates a new DI container.
func NewContainer() *Container {
	return &Container{
		constructors: make(map[reflect.Type]reflect.Value),
		instances:    make(map[reflect.Type]reflect.Value),
		building:     make(map[reflect.Type]bool),
	}
}

// Provide registers a constructor function.
// The constructor should be a function that returns one value and optionally an error.
func (c *Container) Provide(constructor any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	constructorValue := reflect.ValueOf(constructor)
	constructorType := constructorValue.Type()

	if constructorType.Kind() != reflect.Func {
		return fmt.Errorf("constructor must be a function")
	}

	if constructorType.NumOut() == 0 || constructorType.NumOut() > 2 {
		return fmt.Errorf("constructor must return 1 or 2 values")
	}

	// Check if second return value is error
	if constructorType.NumOut() == 2 {
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if !constructorType.Out(1).Implements(errorInterface) {
			return fmt.Errorf("constructor's second return value must be error")
		}
	}

	returnType := constructorType.Out(0)
	c.constructors[returnType] = constructorValue

	return nil
}

// Resolve resolves a dependency and stores it in the provided pointer.
func (c *Container) Resolve(target any) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetType := targetValue.Elem().Type()

	instance, err := c.getInstance(targetType)
	if err != nil {
		return err
	}

	targetValue.Elem().Set(instance)
	return nil
}

// getInstance gets or creates an instance of the given type.
func (c *Container) getInstance(typ reflect.Type) (reflect.Value, error) {
	c.mu.RLock()
	// Check if already built
	if instance, ok := c.instances[typ]; ok {
		c.mu.RUnlock()
		return instance, nil
	}

	// Check for circular dependency
	if c.building[typ] {
		c.mu.RUnlock()
		return reflect.Value{}, fmt.Errorf("circular dependency detected for type %s", typ)
	}
	c.mu.RUnlock()

	// Mark as building
	c.mu.Lock()
	c.building[typ] = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.building, typ)
		c.mu.Unlock()
	}()

	// Get constructor
	c.mu.RLock()
	constructor, ok := c.constructors[typ]
	c.mu.RUnlock()

	if !ok {
		return reflect.Value{}, fmt.Errorf("no constructor registered for type %s", typ)
	}

	// Build dependencies
	constructorType := constructor.Type()
	args := make([]reflect.Value, constructorType.NumIn())

	for i := 0; i < constructorType.NumIn(); i++ {
		paramType := constructorType.In(i)
		paramInstance, err := c.getInstance(paramType)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to resolve dependency %s: %w", paramType, err)
		}
		args[i] = paramInstance
	}

	// Call constructor
	results := constructor.Call(args)

	// Check for error
	if len(results) == 2 {
		if !results[1].IsNil() {
			return reflect.Value{}, fmt.Errorf("constructor failed: %v", results[1].Interface())
		}
	}

	instance := results[0]

	// Cache instance
	c.mu.Lock()
	c.instances[typ] = instance
	c.mu.Unlock()

	return instance, nil
}
