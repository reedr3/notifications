package services_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/notifications/fakes"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/web/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TemplateAssigner", func() {
	var assigner services.TemplateAssigner
	var kindsRepo *fakes.KindsRepo
	var clientsRepo *fakes.ClientsRepo
	var templatesRepo *fakes.TemplatesRepo
	var conn *fakes.DBConn
	var database *fakes.Database

	BeforeEach(func() {
		conn = fakes.NewDBConn()
		database = fakes.NewDatabase()
		clientsRepo = fakes.NewClientsRepo()
		kindsRepo = fakes.NewKindsRepo()
		templatesRepo = fakes.NewTemplatesRepo()
		assigner = services.NewTemplateAssigner(clientsRepo, kindsRepo, templatesRepo, database)
	})

	Describe("AssignToClient", func() {
		BeforeEach(func() {
			var err error

			_, err = clientsRepo.Create(conn, models.Client{
				ID: "my-client",
			})
			if err != nil {
				panic(err)
			}

			_, err = templatesRepo.Create(conn, models.Template{
				ID: "my-template",
			})
			if err != nil {
				panic(err)
			}
		})

		It("assigns the template to the given client", func() {
			err := assigner.AssignToClient("my-client", "my-template")
			Expect(err).NotTo(HaveOccurred())

			client, err := clientsRepo.Find(conn, "my-client")
			if err != nil {
				panic(err)
			}

			Expect(client.Template).To(Equal("my-template"))
		})

		Context("when the request includes a non-existant id", func() {
			It("reports that the client cannot be found", func() {
				err := assigner.AssignToClient("bad-client", "my-template")
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(models.ErrRecordNotFound{}))
			})

			It("reports that the template cannot be found", func() {
				err := assigner.AssignToClient("my-client", "non-existant-template")
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(services.TemplateAssignmentError("")))
			})
		})

		Context("when it gets an error it doesn't understand", func() {
			Context("on finding the client", func() {
				It("returns any errors it doesn't understand", func() {
					clientsRepo.FindError = errors.New("database connection failure")
					err := assigner.AssignToClient("my-client", "my-template")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("database connection failure"))
				})
			})
			Context("on finding the template", func() {
				It("returns any errors it doesn't understand (part 2)", func() {
					templatesRepo.FindError = errors.New("database failure")
					err := assigner.AssignToClient("my-client", "my-template")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("database failure"))

				})
			})

			Context("on updating the client", func() {
				It("Returns the error", func() {
					clientsRepo.UpdateError = errors.New("database fail")
					err := assigner.AssignToClient("my-client", "my-template")
					Expect(err).To(HaveOccurred())
				})
			})

		})
	})

	Describe("AssignToNotification", func() {
		BeforeEach(func() {
			client, err := clientsRepo.Create(conn, models.Client{
				ID: "my-client",
			})
			if err != nil {
				panic(err)
			}

			_, err = kindsRepo.Create(conn, models.Kind{
				ID:       "my-kind",
				ClientID: client.ID,
			})
			if err != nil {
				panic(err)
			}

			_, err = templatesRepo.Create(conn, models.Template{
				ID: "my-template",
			})
			if err != nil {
				panic(err)
			}
		})

		It("assigns the template to the given kind", func() {
			err := assigner.AssignToNotification("my-client", "my-kind", "my-template")
			Expect(err).NotTo(HaveOccurred())

			kind, err := kindsRepo.Find(conn, "my-kind", "my-client")
			if err != nil {
				panic(err)
			}

			Expect(kind.Template).To(Equal("my-template"))
		})

		Context("when the request includes a non-existant id", func() {
			It("reports that the client cannot be found", func() {
				err := assigner.AssignToNotification("bad-client", "my-kind", "my-template")
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(models.ErrRecordNotFound{}))
			})

			It("reports that the kind cannot be found", func() {
				err := assigner.AssignToNotification("my-client", "bad-kind", "my-template")
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(models.ErrRecordNotFound{}))
			})

			It("reports that the template cannot be found", func() {
				err := assigner.AssignToNotification("my-client", "my-kind", "non-existant-template")
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(services.TemplateAssignmentError("")))
			})
		})

		Context("when it gets an error it doesn't understand", func() {
			Context("on finding the client", func() {
				It("returns any errors it doesn't understand", func() {
					clientsRepo.FindError = errors.New("database connection failure")
					err := assigner.AssignToNotification("my-client", "my-kind", "my-template")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("database connection failure"))
				})
			})
			Context("on finding the template", func() {
				It("returns any errors it doesn't understand (part 2)", func() {
					templatesRepo.FindError = errors.New("database failure")
					err := assigner.AssignToNotification("my-client", "my-kind", "my-template")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("database failure"))

				})
			})

			Context("on updating the client", func() {
				It("Returns the error", func() {
					kindsRepo.UpdateError = errors.New("database fail")
					err := assigner.AssignToNotification("my-client", "my-kind", "my-template")
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
