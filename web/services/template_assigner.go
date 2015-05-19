package services

import "github.com/cloudfoundry-incubator/notifications/models"

type TemplateAssignerInterface interface {
	AssignToClient(models.DatabaseInterface, string, string) error
	AssignToNotification(models.DatabaseInterface, string, string, string) error
}

type TemplateAssigner struct {
	clientsRepo   models.ClientsRepoInterface
	kindsRepo     models.KindsRepoInterface
	templatesRepo models.TemplatesRepoInterface
}

func NewTemplateAssigner(clientsRepo models.ClientsRepoInterface,
	kindsRepo models.KindsRepoInterface,
	templatesRepo models.TemplatesRepoInterface) TemplateAssigner {

	return TemplateAssigner{
		clientsRepo:   clientsRepo,
		kindsRepo:     kindsRepo,
		templatesRepo: templatesRepo,
	}
}

func (assigner TemplateAssigner) AssignToClient(database models.DatabaseInterface, clientID, templateID string) error {
	conn := database.Connection()

	if templateID == "" {
		templateID = models.DefaultTemplateID
	}

	client, err := assigner.clientsRepo.Find(conn, clientID)
	if err != nil {
		return err
	}

	err = assigner.findTemplate(conn, templateID)
	if err != nil {
		return err
	}

	client.TemplateID = templateID

	_, err = assigner.clientsRepo.Update(conn, client)
	if err != nil {
		return err
	}

	return nil
}

func (assigner TemplateAssigner) AssignToNotification(database models.DatabaseInterface, clientID, notificationID, templateID string) error {
	conn := database.Connection()

	if templateID == "" {
		templateID = models.DefaultTemplateID
	}

	_, err := assigner.clientsRepo.Find(conn, clientID)
	if err != nil {
		return err
	}

	kind, err := assigner.kindsRepo.Find(conn, notificationID, clientID)
	if err != nil {
		return err
	}

	err = assigner.findTemplate(conn, templateID)
	if err != nil {
		return err
	}

	kind.TemplateID = templateID

	_, err = assigner.kindsRepo.Update(conn, kind)
	if err != nil {
		return err
	}

	return nil
}

func (assigner TemplateAssigner) findTemplate(conn models.ConnectionInterface, templateID string) error {
	if templateID == "" {
		return nil
	}

	_, err := assigner.templatesRepo.FindByID(conn, templateID)
	if err != nil {
		if _, ok := err.(models.RecordNotFoundError); ok {
			return TemplateAssignmentError("No template with id '" + templateID + "'")
		}
		return err
	}

	return nil
}
