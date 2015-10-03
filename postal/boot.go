package postal

import (
	"crypto/rand"
	"database/sql"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/db"
	"github.com/cloudfoundry-incubator/notifications/gobble"
	"github.com/cloudfoundry-incubator/notifications/mail"
	"github.com/cloudfoundry-incubator/notifications/postal/common"
	"github.com/cloudfoundry-incubator/notifications/postal/v1"
	"github.com/cloudfoundry-incubator/notifications/postal/v2"
	"github.com/cloudfoundry-incubator/notifications/strategy"
	"github.com/cloudfoundry-incubator/notifications/uaa"
	"github.com/cloudfoundry-incubator/notifications/v1/models"
	"github.com/cloudfoundry-incubator/notifications/v1/services"
	"github.com/cloudfoundry-incubator/notifications/v2/collections"
	v2models "github.com/cloudfoundry-incubator/notifications/v2/models"
	"github.com/cloudfoundry-incubator/notifications/v2/queue"
	"github.com/pivotal-cf/cf-autoscaling/util"
	"github.com/pivotal-golang/conceal"
	"github.com/pivotal-golang/lager"
)

type mother interface {
	SQLDatabase() *sql.DB
	Database() db.DatabaseInterface
	MailClient() *mail.Client
}

type Config struct {
	UAAClientID          string
	UAAClientSecret      string
	UAAPublicKey         string
	UAAHost              string
	VerifySSL            bool
	InstanceIndex        int
	WorkerCount          int
	EncryptionKey        []byte
	DBLoggingEnabled     bool
	Sender               string
	Domain               string
	QueueWaitMaxDuration int
	CCHost               string
}

func Boot(mom mother, config Config) {
	uaaClient := uaa.NewZonedUAAClient(config.UAAClientID, config.UAAClientSecret, config.VerifySSL, config.UAAPublicKey)

	logger := lager.NewLogger("notifications")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	sqlDatabase := mom.SQLDatabase()
	database := mom.Database()

	gobbleDatabase := gobble.NewDatabase(sqlDatabase)
	gobbleQueue := gobble.NewQueue(gobbleDatabase, gobble.Config{
		WaitMaxDuration: time.Duration(config.QueueWaitMaxDuration) * time.Millisecond,
	})

	cloak, err := conceal.NewCloak(config.EncryptionKey)
	if err != nil {
		panic(err)
	}

	guidGenerator := v2models.NewIDGenerator(rand.Reader)

	// V1
	receiptsRepo := models.NewReceiptsRepo()
	unsubscribesRepo := models.NewUnsubscribesRepo()
	globalUnsubscribesRepo := models.NewGlobalUnsubscribesRepo()
	messagesRepo := models.NewMessagesRepo(guidGenerator.Generate)
	clientsRepo := models.NewClientsRepo()
	kindsRepo := models.NewKindsRepo()
	templatesRepo := models.NewTemplatesRepo()
	v1TemplateLoader := v1.NewTemplatesLoader(database, clientsRepo, kindsRepo, templatesRepo)
	deliveryFailureHandler := common.NewDeliveryFailureHandler()
	messageStatusUpdater := v1.NewMessageStatusUpdater(messagesRepo)
	userLoader := common.NewUserLoader(uaaClient)
	tokenLoader := uaa.NewTokenLoader(uaaClient)
	packager := common.NewPackager(v1TemplateLoader, cloak)

	// V2
	messagesRepository := v2models.NewMessagesRepository(util.NewClock(), guidGenerator.Generate)

	v1enqueuer := services.NewEnqueuer(gobbleQueue, messagesRepo)
	v2enqueuer := queue.NewJobEnqueuer(gobbleQueue, messagesRepository)

	cloudController := cf.NewCloudController(config.CCHost, !config.VerifySSL)
	spaceLoader := services.NewSpaceLoader(cloudController)
	organizationLoader := services.NewOrganizationLoader(cloudController)
	findsUserIDs := services.NewFindsUserIDs(cloudController, uaaClient)

	orgStrategy := services.NewOrganizationStrategy(tokenLoader, organizationLoader, findsUserIDs, v1enqueuer, v2enqueuer)
	spaceStrategy := services.NewSpaceStrategy(tokenLoader, spaceLoader, organizationLoader, findsUserIDs, v1enqueuer, v2enqueuer)
	userStrategy := services.NewUserStrategy(v1enqueuer, v2enqueuer)
	emailStrategy := services.NewEmailStrategy(v1enqueuer, v2enqueuer)
	v2database := v2models.NewDatabase(sqlDatabase, v2models.Config{})
	v2messageStatusUpdater := v2.NewV2MessageStatusUpdater(messagesRepository)
	unsubscribersRepository := v2models.NewUnsubscribersRepository(guidGenerator.Generate)
	campaignsRepository := v2models.NewCampaignsRepository(guidGenerator.Generate)
	v2templatesRepo := v2models.NewTemplatesRepository(guidGenerator.Generate)
	templatesCollection := collections.NewTemplatesCollection(v2templatesRepo)
	v2TemplateLoader := v2.NewTemplatesLoader(v2database, templatesCollection)
	v2deliveryFailureHandler := common.NewDeliveryFailureHandler()
	strategyDeterminer := strategy.NewStrategyDeterminer(userStrategy, spaceStrategy, orgStrategy, emailStrategy)

	WorkerGenerator{
		InstanceIndex: config.InstanceIndex,
		Count:         config.WorkerCount,
	}.Work(func(index int) Worker {

		mailClient := mom.MailClient()

		v1Workflow := v1.NewProcess(v1.ProcessConfig{
			DBTrace: config.DBLoggingEnabled,
			UAAHost: config.UAAHost,
			Sender:  config.Sender,
			Domain:  config.Domain,

			Packager:    packager,
			MailClient:  mailClient,
			Database:    database,
			TokenLoader: tokenLoader,
			UserLoader:  userLoader,

			KindsRepo:              kindsRepo,
			ReceiptsRepo:           receiptsRepo,
			UnsubscribesRepo:       unsubscribesRepo,
			GlobalUnsubscribesRepo: globalUnsubscribesRepo,
			MessageStatusUpdater:   messageStatusUpdater,
			DeliveryFailureHandler: deliveryFailureHandler,
		})

		v2mailClient := mom.MailClient()

		v2Workflow := v2.NewWorkflow(v2mailClient, common.NewPackager(v2TemplateLoader, cloak),
			common.NewUserLoader(uaaClient), uaa.NewTokenLoader(uaaClient), v2messageStatusUpdater, v2database,
			unsubscribersRepository, campaignsRepository, config.Sender, config.Domain, config.UAAHost)

		worker := NewDeliveryWorker(v1Workflow, v2Workflow, DeliveryWorkerConfig{
			ID:      index,
			UAAHost: config.UAAHost,
			DBTrace: config.DBLoggingEnabled,

			Logger: logger,
			Queue:  gobbleQueue,

			Database:               v2database,
			StrategyDeterminer:     strategyDeterminer,
			DeliveryFailureHandler: v2deliveryFailureHandler,
			MessageStatusUpdater:   v2messageStatusUpdater,
		})

		return &worker
	})
}
