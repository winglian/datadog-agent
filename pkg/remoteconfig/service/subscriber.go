package service

type SubscriberUpdate struct {
	RootVersion uint64
	Targets     []byte
}

type Subscriber interface {
	Notify(update SubscriberUpdate) error
}
