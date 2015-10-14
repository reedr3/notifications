package v2

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/gobble"
	"github.com/cloudfoundry-incubator/notifications/v1/services"
	"github.com/cloudfoundry-incubator/notifications/v2/horde"
	"github.com/cloudfoundry-incubator/notifications/v2/queue"
)

type NoAudienceError struct {
	Err error
}

func (e NoAudienceError) Error() string {
	return e.Err.Error()
}

type recipient struct {
	Email string
	GUID  string
}

type emailAddressFormatter interface {
	Format(email string) (formattedEmail string)
}

type htmlPartsExtractor interface {
	Extract(html string) (doctype, head, bodyContent, bodyAttributes string, err error)
}

type CampaignJobProcessor struct {
	emailFormatter emailAddressFormatter
	htmlExtractor  htmlPartsExtractor
	enqueuer       enqueuer

	emails audienceGenerator
	spaces audienceGenerator
	orgs   audienceGenerator
	users  audienceGenerator
}

type audienceGenerator interface {
	GenerateAudiences(inputs []string) ([]horde.Audience, error)
}

type enqueuer interface {
	Enqueue(conn queue.ConnectionInterface, users []queue.User, options queue.Options, space cf.CloudControllerSpace, organization cf.CloudControllerOrganization, clientID, uaaHost, scope, vcapRequestID string, reqReceived time.Time, campaignID string)
}

func NewCampaignJobProcessor(emailFormatter emailAddressFormatter, htmlExtractor htmlPartsExtractor, emails, spaces, orgs, users audienceGenerator, enqueuer enqueuer) CampaignJobProcessor {
	return CampaignJobProcessor{
		emailFormatter: emailFormatter,
		htmlExtractor:  htmlExtractor,
		enqueuer:       enqueuer,
		emails:         emails,
		spaces:         spaces,
		orgs:           orgs,
		users:          users,
	}
}

func (p CampaignJobProcessor) Process(conn services.ConnectionInterface, uaaHost string, job gobble.Job) error {
	var campaignJob queue.CampaignJob

	err := job.Unmarshal(&campaignJob)
	if err != nil {
		return err
	}

	doctype, head, bodyContent, bodyAttributes, err := p.htmlExtractor.Extract(campaignJob.Campaign.HTML)
	if err != nil {
		return err
	}

	var audiences []horde.Audience
	for audienceName, audienceMembers := range campaignJob.Campaign.SendTo {
		generator, err := p.findAudienceGenerator(audienceName)
		if err != nil {
			return err
		}

		aud, err := generator.GenerateAudiences(audienceMembers)
		if err != nil {
			return err
		}

		audiences = append(audiences, aud...)
	}

	for _, audience := range audiences {
		var users []queue.User
		for _, user := range audience.Users {
			users = append(users, queue.User{
				GUID:  user.GUID,
				Email: user.Email,
			})
		}

		options := queue.Options{
			ReplyTo: campaignJob.Campaign.ReplyTo,
			Subject: campaignJob.Campaign.Subject,
			Text:    campaignJob.Campaign.Text,
			HTML: queue.HTML{
				Doctype:        doctype,
				Head:           head,
				BodyContent:    bodyContent,
				BodyAttributes: bodyAttributes,
			},
			Endorsement: audience.Endorsement,
			TemplateID:  campaignJob.Campaign.TemplateID,
		}

		p.enqueuer.Enqueue(conn, users, options, cf.CloudControllerSpace{},
			cf.CloudControllerOrganization{}, campaignJob.Campaign.ClientID,
			uaaHost, "", "", time.Time{}, campaignJob.Campaign.ID)
	}

	return nil
}

func (p CampaignJobProcessor) findAudienceGenerator(audience string) (audienceGenerator, error) {
	switch audience {
	case "users":
		return p.users, nil
	case "spaces":
		return p.spaces, nil
	case "orgs":
		return p.orgs, nil
	case "emails":
		return p.emails, nil
	default:
		return nil, NoAudienceError{fmt.Errorf("generator for %q audience could not be found", audience)}
	}
}
