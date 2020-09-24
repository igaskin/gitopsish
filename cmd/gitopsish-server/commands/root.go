/*
Copyright © 2020 Isaac Gaskin

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const (
	targetUser = "igaskin"
)

type feelings struct {
	srv            *http.Server
	friendship     map[string]*github.User
	stateToCodeMap map[string]string
}

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitopsish-server",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		var wait time.Duration
		ctx, cancel := context.WithTimeout(context.Background(), wait)
		r := mux.NewRouter()
		feelme := feelings{
			srv: &http.Server{
				Addr:         "127.0.0.1:9999",
				WriteTimeout: time.Second * 15,
				ReadTimeout:  time.Second * 15,
				IdleTimeout:  time.Second * 60,
				Handler:      r, // Pass our instance of gorilla/mux in.
			},
			friendship:     make(map[string]*github.User),
			stateToCodeMap: make(map[string]string),
		}
		// Add your routes as needed
		r.HandleFunc("/", feelme.register)
		r.HandleFunc("/are-you-ok", feelme.okayish)
		r.HandleFunc("/callback", feelme.callback)

		go func() {
			log.Infof("starting webserver on %v", feelme.srv.Addr)
			if err := feelme.srv.ListenAndServe(); err != nil {
				log.Println(err)
			}
		}()
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		<-c

		defer cancel()
		err := feelme.srv.Shutdown(ctx)
		if err != nil {
			log.Fatal("failed graceful shutdown")
		}
		log.Println("shutting down")
		os.Exit(0)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gitopsish-server.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gitopsish-server" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".gitopsish-server")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func (f *feelings) okayish(w http.ResponseWriter, r *http.Request) {
	if ok := r.URL.Query().Get("really"); ok == "true" {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte("permission denied"))
		if err != nil {
			log.Info("failed to response")
		}
	} else {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			log.Info("failed to response")
		}
	}
}

func (f *feelings) callback(w http.ResponseWriter, r *http.Request) {
	if code := r.URL.Query().Get("code"); code != "" {
		req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", nil)
		q := req.URL.Query()
		q.Add("client_id", os.Getenv("CLIENT_ID"))
		q.Add("client_secret", os.Getenv("CLIENT_SECRET"))
		q.Add("code", code)
		q.Add("redirect_uri", "http://localhost:9999/callback")
		q.Add("state", r.URL.Query().Get("state")) // figure out how to store this state
		req.URL.RawQuery = q.Encode()

		res, _ := http.DefaultClient.Do(req)

		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Warn("unable to read response body")
		}
		responseQuery, err := url.ParseQuery(string(body))
		if err != nil {
			log.Warn("unable to read response query")
		}
		if token := responseQuery.Get("access_token"); token != "" {
			log.Info("sucessfully authenticated user")
			ctx := context.Background()
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: token},
			)
			tc := oauth2.NewClient(ctx, ts)
			gh := github.NewClient(tc)
			isFollowing, _, err := gh.Users.IsFollowing(ctx, "", targetUser)
			if err != nil {
				log.Warn("unable to check following status")
			}
			currentUser, _, _ := gh.Users.Get(ctx, "")
			if isFollowing {
				log.Infof("%s is following me!", *currentUser.Login)
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("existance is torment"))
				if err != nil {
					log.Info("failed to respond")
				}
			} else {
				log.Info()
				w.WriteHeader(http.StatusPreconditionFailed)
				_, err := w.Write([]byte(fmt.Sprintf("%s, please follow me first here: https://github.com/%s", *currentUser.Login, targetUser)))
				if err != nil {
					log.Info("failed to respond")
				}
			}

		}
		responseParams, err := url.ParseQuery(string(body))
		if err != nil {
			log.Warn("unable to parse oauth response")
		}
		if err := responseParams.Get("error"); err != "" {
			log.Warn(responseParams)
		}
		log.Info(res.Status)
	}
}

func (f *feelings) register(w http.ResponseWriter, r *http.Request) {
	// TODO(igaskin) add cookies to maintain state
	req, _ := http.NewRequest(http.MethodGet, "https://github.com/login/oauth/authorize", nil)
	q := req.URL.Query()
	state := string(uuid.NewUUID())
	f.stateToCodeMap[state] = ""
	q.Add("client_id", os.Getenv("CLIENT_ID"))
	q.Add("redirect_uri", "http://localhost:9999/callback")
	q.Add("scope", "read:user")
	q.Add("state", state)
	req.URL.RawQuery = q.Encode()

	http.Redirect(w, r, req.URL.String(), http.StatusSeeOther)
}
