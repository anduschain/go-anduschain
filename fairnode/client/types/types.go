package types

type Goroutine struct {
	Fn   func(exit chan struct{}, v interface{})
	Exit chan struct{}
}
