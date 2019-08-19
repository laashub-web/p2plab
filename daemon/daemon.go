// Copyright 2019 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type Daemon struct {
	addr   string
	router *mux.Router
	logger zerolog.Logger
}

func New(addr string, routers ...Router) *Daemon {
	d := &Daemon{
		addr:   addr,
		logger: zerolog.New(os.Stderr).With().Timestamp().Logger(),
	}
	d.router = d.createMux(routers...)
	return d
}

func (d *Daemon) Serve(ctx context.Context) error {
	d.logger.Info().Msgf("daemon listening on %s", d.addr)
	s := &http.Server{
		Handler:     d.router,
		Addr:        d.addr,
		ReadTimeout: 10 * time.Second,
	}
	return s.ListenAndServe()
}

func (d *Daemon) createMux(routers ...Router) *mux.Router {
	d.router = mux.NewRouter().UseEncodedPath().StrictSlash(true)
	for _, router := range routers {
		for _, route := range router.Routes() {
			d.logger.Debug().Str("path", route.Path()).Str("method", route.Method()).Msg("Registering route")
			h := d.createHTTPHandler(route.Handler())
			d.router.Path(route.Path()).Methods(route.Method()).Handler(h)
		}
	}
	return d.router
}

func (d *Daemon) createHTTPHandler(handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := d.logger.WithContext(r.Context())

		r = r.WithContext(ctx)

		vars := mux.Vars(r)
		if vars == nil {
			vars = make(map[string]string)
		}

		err := handler(ctx, w, r, vars)
		if err != nil {
			d.logger.Debug().Err(err).Msg("failed request")
			if errdefs.IsAlreadyExists(err) {
				http.Error(w, err.Error(), http.StatusConflict)
			} else if errdefs.IsNotFound(err) {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else if errdefs.IsInvalidArgument(err) {
				http.Error(w, err.Error(), http.StatusNotAcceptable)
			} else if errdefs.IsUnavailable(err) {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
			} else {
				// Any error types we don't specifically look out for default to serving a
				// HTTP 500.
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}

func WriteJSON(w http.ResponseWriter, v interface{}) error {
	content, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err
	}
	w.Write(content)
	return nil
}
