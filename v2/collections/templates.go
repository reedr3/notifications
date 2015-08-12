package collections

import (
	"github.com/cloudfoundry-incubator/notifications/db"
	"github.com/cloudfoundry-incubator/notifications/v2/models"
)

type Template struct {
	ID       string
	Name     string
	HTML     string
	Text     string
	Subject  string
	Metadata string
	ClientID string
}

type templatesRepository interface {
	Insert(conn db.ConnectionInterface, template models.Template) (createdTemplate models.Template, err error)
	Get(conn db.ConnectionInterface, templateID string) (retrievedTemplate models.Template, err error)
}

type TemplatesCollection struct {
	repo templatesRepository
}

func NewTemplatesCollection(repo templatesRepository) TemplatesCollection {
	return TemplatesCollection{
		repo: repo,
	}
}

func (c TemplatesCollection) Set(conn ConnectionInterface, template Template) (createdTemplate Template, err error) {
	model, err := c.repo.Insert(conn, models.Template{
		Name:     template.Name,
		HTML:     template.HTML,
		Text:     template.Text,
		Subject:  template.Subject,
		Metadata: template.Metadata,
		ClientID: template.ClientID,
	})
	if err != nil {
		panic(err)
	}

	return Template{
		ID:       model.ID,
		Name:     model.Name,
		HTML:     model.HTML,
		Text:     model.Text,
		Subject:  model.Subject,
		Metadata: model.Metadata,
		ClientID: model.ClientID,
	}, nil
}

func (c TemplatesCollection) Get(conn ConnectionInterface, templateID, clientID string) (Template, error) {
	model, err := c.repo.Get(conn, templateID)
	if err != nil {
		panic(err)
	}
	return Template{
		ID:       model.ID,
		Name:     model.Name,
		HTML:     model.HTML,
		Text:     model.Text,
		Subject:  model.Subject,
		Metadata: model.Metadata,
		ClientID: model.ClientID,
	}, nil
}
