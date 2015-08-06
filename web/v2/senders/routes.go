package senders

import (
	"github.com/cloudfoundry-incubator/notifications/collections"
	"github.com/ryanmoran/stack"
)

type muxer interface {
	Handle(method, path string, handler stack.Handler, middleware ...stack.Middleware)
}

type Routes struct {
	RequestLogging    stack.Middleware
	Authenticator     stack.Middleware
	DatabaseAllocator stack.Middleware
	SendersCollection collections.SendersCollection
}

func (r Routes) Register(m muxer) {
	m.Handle("POST", "/senders", NewCreateHandler(r.SendersCollection), r.RequestLogging, r.Authenticator, r.DatabaseAllocator)
	m.Handle("GET", "/senders/{sender_id}", NewGetHandler(r.SendersCollection), r.RequestLogging, r.Authenticator, r.DatabaseAllocator)
}