package types

type Goroutine struct {
	Fn   func(exit chan struct{})
	Exit chan struct{}
}
