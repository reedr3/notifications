package handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/notifications/metrics"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal"
	"github.com/cloudfoundry-incubator/notifications/postal/strategies"
	"github.com/ryanmoran/stack"
)

type NotifyEmail struct {
	errorWriter ErrorWriterInterface
	notify      NotifyInterface
	strategy    strategies.StrategyInterface
	database    models.DatabaseInterface
}

func NewNotifyEmail(notify NotifyInterface, errorWriter ErrorWriterInterface, strategy strategies.StrategyInterface, database models.DatabaseInterface) NotifyEmail {
	return NotifyEmail{
		errorWriter: errorWriter,
		notify:      notify,
		strategy:    strategy,
		database:    database,
	}
}

func (handler NotifyEmail) ServeHTTP(w http.ResponseWriter, req *http.Request, context stack.Context) {
	connection := handler.database.Connection()
	err := handler.Execute(w, req, connection, context)
	if err != nil {
		handler.errorWriter.Write(w, err)
		return
	}

	metrics.NewMetric("counter", map[string]interface{}{
		"name": "notifications.web.emails",
	}).Log()
}

func (handler NotifyEmail) Execute(w http.ResponseWriter, req *http.Request, connection models.ConnectionInterface, context stack.Context) error {
	output, err := handler.notify.Execute(connection, req, context, postal.EmailID(""), handler.strategy)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	w.Write(output)
	return nil
}
