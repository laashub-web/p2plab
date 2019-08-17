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

package controlapi

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/pkg/httputil"
)

type scenarioAPI struct {
	client *httputil.Client
	url    urlFunc
}

func (a *scenarioAPI) Create(ctx context.Context, id string, sdef metadata.ScenarioDefinition) (p2plab.Scenario, error) {
	content, err := json.MarshalIndent(&sdef, "", "    ")
	if err != nil {
		return nil, err
	}

	req := a.client.NewRequest("POST", a.url("/scenarios")).
		Option("id", id).
		Body(bytes.NewReader(content))

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	s := scenario{client: a.client}
	err = json.NewDecoder(resp.Body).Decode(&s.metadata)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (a *scenarioAPI) Get(ctx context.Context, name string) (p2plab.Scenario, error) {
	req := a.client.NewRequest("GET", a.url("/scenarios/%s", name))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	s := scenario{client: a.client, url: a.url}
	err = json.NewDecoder(resp.Body).Decode(&s.metadata)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (a *scenarioAPI) List(ctx context.Context) ([]p2plab.Scenario, error) {
	req := a.client.NewRequest("GET", a.url("/scenarios"))
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metadatas []metadata.Scenario
	err = json.NewDecoder(resp.Body).Decode(&metadatas)
	if err != nil {
		return nil, err
	}

	var scenarios []p2plab.Scenario
	for _, m := range metadatas {
		scenarios = append(scenarios, &scenario{
			client:   a.client,
			metadata: m,
			url:      a.url,
		})
	}

	return scenarios, nil
}

type scenario struct {
	client   *httputil.Client
	metadata metadata.Scenario
	url      urlFunc
}

func (s *scenario) Metadata() metadata.Scenario {
	return s.metadata
}

func (s *scenario) Remove(ctx context.Context) error {
	req := s.client.NewRequest("DELETE", s.url("/scenarios/%s", s.metadata.ID))
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}