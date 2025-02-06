package slackbot

import (
	"fmt"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/variablefinders"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/services"
	"github.com/jenkins-x/jx-helpers/v3/pkg/requirements"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/slack-go/slack"
	"k8s.io/apimachinery/pkg/util/runtime"
)

// Validate configures the clients for the slack bot
func (o *Options) Validate() error {
	if o.SlackClient == nil {
		if o.SlackToken == "" {
			return fmt.Errorf("no $SLACK_TOKEN defined")
		}
		if o.SlackURL != "" {
			log.Logger().Infof("using slack URL %s", o.SlackURL)
			o.SlackClient = slack.New(o.SlackToken, slack.OptionAPIURL(o.SlackURL))
		} else {
			o.SlackClient = slack.New(o.SlackToken)
		}
	}

	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return err
	}

	o.JXClient, err = jxclient.LazyCreateJXClient(o.JXClient)
	if err != nil {
		return err
	}

	if o.ScmClient == nil {
		o.ScmClient, err = factory.NewClientFromEnvironment()
		if err != nil {
			return fmt.Errorf("failed to create SCM client: %w", err)
		}
	}
	o.SlackUserResolver = NewSlackUserResolver(o.SlackClient, o.JXClient, o.Namespace)

	if o.Dir == "" {
		if o.GitClient == nil {
			o.GitClient = cli.NewCLIClient("", o.CommandRunner)
		}

		if o.GitURL == "" {
			req, err := variablefinders.FindRequirements(o.GitClient, o.JXClient, o.Namespace, o.Dir, "", "")
			if err != nil {
				return fmt.Errorf("failed to load requirements from dev environment: %w", err)
			}

			if req == nil {
				return fmt.Errorf("no Requirements in TeamSettings of dev environment in namespace %s", o.Namespace)
			}

			// lets override the dev git URL if its changed in the requirements via the .jx/settings.yaml file
			o.GitURL = requirements.EnvironmentGitURL(req, "dev")
			if o.GitURL == "" {
				return fmt.Errorf("could not find development environment git URL from requirements and no $GIT_URL specified")
			}
		}
		o.Dir, err = gitclient.CloneToDir(o.GitClient, o.GitURL, "")
		if err != nil {
			return fmt.Errorf("failed to clone git URL %s: %w", o.GitURL, err)
		}
	}
	o.SourceConfigs, err = sourceconfigs.LoadSourceConfig(o.Dir, true)
	if err != nil {
		return fmt.Errorf("failed to load source configs from dir %s: %w", o.Dir, err)
	}

	// lets find the dashboard URL
	if o.MessageFormat.DashboardURL == "" {
		ingressName := "jx-pipelines-visualizer"
		o.MessageFormat.DashboardURL, err = services.FindIngressURL(o.KubeClient, o.Namespace, ingressName)
		if err != nil {
			return fmt.Errorf("failed to find dashboard ingress %s in namespace %s: %w", ingressName, o.Namespace, err)
		}

		if o.MessageFormat.DashboardURL == "" {
			log.Logger().Warnf("no dashboard ingress %s in namespace %s so cannot link to the dasboard", ingressName, o.Namespace)
		}

	}
	if o.MessageFormat.DashboardURL != "" {
		log.Logger().Infof("using the dashboard URL %s", o.MessageFormat.DashboardURL)
	}
	return nil
}

func (o *Options) Run() error {
	defer runtime.HandleCrash()

	err := o.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate options: %w", err)
	}

	log.Logger().Infof("Watching slackbots in namespace %s\n", o.Namespace)

	o.WatchActivities()
	return nil
}
