package serviceaccess_test

import (
	"errors"

	"github.com/cloudfoundry/cli/cf/actors/actorsfakes"
	"github.com/cloudfoundry/cli/cf/api/authentication/authenticationfakes"
	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/configuration/coreconfig"
	testcmd "github.com/cloudfoundry/cli/testhelpers/commands"
	"github.com/cloudfoundry/cli/testhelpers/configuration"
	. "github.com/cloudfoundry/cli/testhelpers/matchers"
	testreq "github.com/cloudfoundry/cli/testhelpers/requirements"
	testterm "github.com/cloudfoundry/cli/testhelpers/terminal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("disable-service-access command", func() {
	var (
		ui                  *testterm.FakeUI
		actor               *actorsfakes.FakeServicePlanActor
		requirementsFactory *testreq.FakeReqFactory
		tokenRefresher      *authenticationfakes.FakeRepository
		configRepo          coreconfig.Repository
		deps                commandregistry.Dependency

		serviceName           string
		servicePlanName       string
		publicServicePlanName string
		orgName               string
	)

	updateCommandDependency := func(pluginCall bool) {
		deps.UI = ui
		deps.RepoLocator = deps.RepoLocator.SetAuthenticationRepository(tokenRefresher)
		deps.ServicePlanHandler = actor
		deps.Config = configRepo
		commandregistry.Commands.SetCommand(commandregistry.Commands.FindCommand("disable-service-access").SetDependency(deps, pluginCall))
	}

	BeforeEach(func() {
		ui = &testterm.FakeUI{
			Inputs: []string{"yes"},
		}
		configRepo = configuration.NewRepositoryWithDefaults()
		actor = new(actorsfakes.FakeServicePlanActor)
		requirementsFactory = &testreq.FakeReqFactory{}
		tokenRefresher = new(authenticationfakes.FakeRepository)
	})

	runCommand := func(args []string) bool {
		return testcmd.RunCLICommand("disable-service-access", args, requirementsFactory, updateCommandDependency, false, ui)
	}

	Describe("requirements", func() {
		It("requires the user to be logged in", func() {
			Expect(runCommand([]string{"foo"})).To(BeFalse())
		})

		It("fails with usage when it does not recieve any arguments", func() {
			requirementsFactory.LoginSuccess = true
			runCommand(nil)
			Expect(ui.Outputs).To(ContainSubstrings(
				[]string{"Incorrect Usage", "Requires", "argument"},
			))
		})
	})

	Describe("when logged in", func() {
		BeforeEach(func() {
			requirementsFactory.LoginSuccess = true

			serviceName = "service"
			servicePlanName = "service-plan"
			publicServicePlanName = "public-service-plan"
			orgName = "my-org"
		})

		It("refreshes the auth token", func() {
			runCommand([]string{serviceName})
			Expect(tokenRefresher.RefreshAuthTokenCallCount()).To(Equal(1))
		})

		Context("when refreshing the auth token fails", func() {
			It("fails and returns the error", func() {
				tokenRefresher.RefreshAuthTokenReturns("", errors.New("Refreshing went wrong"))
				runCommand([]string{serviceName})

				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Refreshing went wrong"},
					[]string{"FAILED"},
				))
			})
		})

		Context("when the named service exists", func() {
			It("disables the service", func() {
				Expect(runCommand([]string{serviceName})).To(BeTrue())
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"OK"},
				))

				Expect(actor.UpdateAllPlansForServiceCallCount()).To(Equal(1))
				service, disable := actor.UpdateAllPlansForServiceArgsForCall(0)
				Expect(service).To(Equal(serviceName))
				Expect(disable).To(BeFalse())
			})

			It("prints an error if updating the plans fails", func() {
				actor.UpdateAllPlansForServiceReturns(errors.New("Kaboom!"))

				Expect(runCommand([]string{serviceName})).To(BeFalse())
				Expect(ui.Outputs).To(ContainSubstrings(
					[]string{"Kaboom!"},
				))
			})

			Context("The user provides a plan", func() {
				It("prints an error if updating the plan fails", func() {
					actor.UpdateSinglePlanForServiceReturns(errors.New("could not find service"))

					Expect(runCommand([]string{"-p", servicePlanName, serviceName})).To(BeFalse())
					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"could not find service"},
					))
				})

				It("disables the plan", func() {
					Expect(runCommand([]string{"-p", publicServicePlanName, serviceName})).To(BeTrue())
					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"OK"},
					))

					Expect(actor.UpdateSinglePlanForServiceCallCount()).To(Equal(1))
					service, plan, disable := actor.UpdateSinglePlanForServiceArgsForCall(0)
					Expect(service).To(Equal(serviceName))
					Expect(plan).To(Equal(publicServicePlanName))
					Expect(disable).To(BeFalse())
				})
			})

			Context("the user provides an org", func() {
				It("prints an error if updating the plan fails", func() {
					actor.UpdateOrgForServiceReturns(errors.New("could not find org"))

					Expect(runCommand([]string{"-o", "not-findable-org", serviceName})).To(BeFalse())
					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"could not find org"},
					))
				})

				It("disables the service for that org", func() {
					Expect(runCommand([]string{"-o", orgName, serviceName})).To(BeTrue())
					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"OK"},
					))

					Expect(actor.UpdateOrgForServiceCallCount()).To(Equal(1))
					service, org, disable := actor.UpdateOrgForServiceArgsForCall(0)
					Expect(service).To(Equal(serviceName))
					Expect(org).To(Equal(orgName))
					Expect(disable).To(BeFalse())
				})
			})

			Context("the user provides a plan and org", func() {
				It("prints an error if updating the plan fails", func() {
					actor.UpdatePlanAndOrgForServiceReturns(errors.New("could not find org"))

					Expect(runCommand([]string{"-p", servicePlanName, "-o", "not-findable-org", serviceName})).To(BeFalse())
					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"could not find org"},
					))
				})

				It("disables the service plan for the org", func() {
					Expect(runCommand([]string{"-p", publicServicePlanName, "-o", orgName, serviceName})).To(BeTrue())
					Expect(ui.Outputs).To(ContainSubstrings(
						[]string{"OK"},
					))

					Expect(actor.UpdatePlanAndOrgForServiceCallCount()).To(Equal(1))
					service, plan, org, disable := actor.UpdatePlanAndOrgForServiceArgsForCall(0)
					Expect(service).To(Equal(serviceName))
					Expect(plan).To(Equal(publicServicePlanName))
					Expect(org).To(Equal(orgName))
					Expect(disable).To(BeFalse())
				})
			})
		})
	})
})
