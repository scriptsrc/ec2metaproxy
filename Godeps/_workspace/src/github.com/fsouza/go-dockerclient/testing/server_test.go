// Copyright 2014 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testing

import (
	"encoding/json"
	"fmt"
	"github.com/flyinprogrammer/ec2metaproxy/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	server, err := NewServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.listener.Close()
	conn, err := net.Dial("tcp", server.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestServerStop(t *testing.T) {
	server, err := NewServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	server.Stop()
	_, err = net.Dial("tcp", server.listener.Addr().String())
	if err == nil {
		t.Error("Unexpected <nil> error when dialing to stopped server")
	}
}

func TestServerStopNoListener(t *testing.T) {
	server := DockerServer{}
	server.Stop()
}

func TestServerURL(t *testing.T) {
	server, err := NewServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Stop()
	url := server.URL()
	if expected := "http://" + server.listener.Addr().String() + "/"; url != expected {
		t.Errorf("DockerServer.URL(): Want %q. Got %q.", expected, url)
	}
}

func TestServerURLNoListener(t *testing.T) {
	server := DockerServer{}
	url := server.URL()
	if url != "" {
		t.Errorf("DockerServer.URL(): Expected empty URL on handler mode, got %q.", url)
	}
}

func TestHandleWithHook(t *testing.T) {
	var called bool
	server, _ := NewServer(func(*http.Request) { called = true })
	defer server.Stop()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/containers/json?all=1", nil)
	server.ServeHTTP(recorder, request)
	if !called {
		t.Error("ServeHTTP did not call the hook function.")
	}
}

func TestListContainers(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 2)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/containers/json?all=1", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("ListContainers: wrong status. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
	expected := make([]docker.APIContainers, 2)
	for i, container := range server.containers {
		expected[i] = docker.APIContainers{
			ID:      container.ID,
			Image:   container.Image,
			Command: strings.Join(container.Config.Cmd, " "),
			Created: container.Created.Unix(),
			Status:  container.State.String(),
			Ports:   container.NetworkSettings.PortMappingAPI(),
		}
	}
	var got []docker.APIContainers
	err := json.NewDecoder(recorder.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ListContainers. Want %#v. Got %#v.", expected, got)
	}
}

func TestListRunningContainers(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 2)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/containers/json?all=0", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("ListRunningContainers: wrong status. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
	var got []docker.APIContainers
	err := json.NewDecoder(recorder.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Errorf("ListRunningContainers: Want 0. Got %d.", len(got))
	}
}

func TestCreateContainer(t *testing.T) {
	server := DockerServer{}
	server.imgIDs = map[string]string{"base": "a1234"}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	body := `{"Hostname":"", "User":"", "Memory":0, "MemorySwap":0, "AttachStdin":false, "AttachStdout":true, "AttachStderr":true,
"PortSpecs":null, "Tty":false, "OpenStdin":false, "StdinOnce":false, "Env":null, "Cmd":["date"],
"Dns":null, "Image":"base", "Volumes":{}, "VolumesFrom":""}`
	request, _ := http.NewRequest("POST", "/containers/create", strings.NewReader(body))
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Errorf("CreateContainer: wrong status. Want %d. Got %d.", http.StatusCreated, recorder.Code)
	}
	var returned docker.Container
	err := json.NewDecoder(recorder.Body).Decode(&returned)
	if err != nil {
		t.Fatal(err)
	}
	stored := server.containers[0]
	if returned.ID != stored.ID {
		t.Errorf("CreateContainer: ID mismatch. Stored: %q. Returned: %q.", stored.ID, returned.ID)
	}
	if stored.State.Running {
		t.Errorf("CreateContainer should not set container to running state.")
	}
}

func TestCreateContainerInvalidBody(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/containers/create", strings.NewReader("whaaaaaat---"))
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("CreateContainer: wrong status. Want %d. Got %d.", http.StatusBadRequest, recorder.Code)
	}
}

func TestCreateContainerImageNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	body := `{"Hostname":"", "User":"", "Memory":0, "MemorySwap":0, "AttachStdin":false, "AttachStdout":true, "AttachStderr":true,
"PortSpecs":null, "Tty":false, "OpenStdin":false, "StdinOnce":false, "Env":null, "Cmd":["date"],
"Dns":null, "Image":"base", "Volumes":{}, "VolumesFrom":""}`
	request, _ := http.NewRequest("POST", "/containers/create", strings.NewReader(body))
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("CreateContainer: wrong status. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func TestCommitContainer(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 2)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/commit?container="+server.containers[0].ID, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("CommitContainer: wrong status. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
	expected := fmt.Sprintf(`{"ID":"%s"}`, server.images[0].ID)
	if got := recorder.Body.String(); got != expected {
		t.Errorf("CommitContainer: wrong response body. Want %q. Got %q.", expected, got)
	}
}

func TestCommitContainerComplete(t *testing.T) {
	server := DockerServer{}
	server.imgIDs = make(map[string]string)
	addContainers(&server, 2)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	queryString := "container=" + server.containers[0].ID + "&repo=tsuru/python&m=saving&author=developers"
	queryString += `&run={"Cmd": ["cat", "/world"],"PortSpecs":["22"]}`
	request, _ := http.NewRequest("POST", "/commit?"+queryString, nil)
	server.ServeHTTP(recorder, request)
	image := server.images[0]
	if image.Parent != server.containers[0].Image {
		t.Errorf("CommitContainer: wrong parent image. Want %q. Got %q.", server.containers[0].Image, image.Parent)
	}
	if image.Container != server.containers[0].ID {
		t.Errorf("CommitContainer: wrong container. Want %q. Got %q.", server.containers[0].ID, image.Container)
	}
	message := "saving"
	if image.Comment != message {
		t.Errorf("CommitContainer: wrong comment (commit message). Want %q. Got %q.", message, image.Comment)
	}
	author := "developers"
	if image.Author != author {
		t.Errorf("CommitContainer: wrong author. Want %q. Got %q.", author, image.Author)
	}
	if id := server.imgIDs["tsuru/python"]; id != image.ID {
		t.Errorf("CommitContainer: wrong ID saved for repository. Want %q. Got %q.", image.ID, id)
	}
	portSpecs := []string{"22"}
	if !reflect.DeepEqual(image.Config.PortSpecs, portSpecs) {
		t.Errorf("CommitContainer: wrong port spec in config. Want %#v. Got %#v.", portSpecs, image.Config.PortSpecs)
	}
	cmd := []string{"cat", "/world"}
	if !reflect.DeepEqual(image.Config.Cmd, cmd) {
		t.Errorf("CommitContainer: wrong cmd in config. Want %#v. Got %#v.", cmd, image.Config.Cmd)
	}
}

func TestCommitContainerInvalidRun(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/commit?container="+server.containers[0].ID+"&run=abc---", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("CommitContainer. Wrong status. Want %d. Got %d.", http.StatusBadRequest, recorder.Code)
	}
}

func TestCommitContainerNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/commit?container=abc123", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("CommitContainer. Wrong status. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func TestInspectContainer(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 2)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s/json", server.containers[0].ID)
	request, _ := http.NewRequest("GET", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("InspectContainer: wrong status. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
	expected := server.containers[0]
	var got docker.Container
	err := json.NewDecoder(recorder.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Config, expected.Config) {
		t.Errorf("InspectContainer: wrong value. Want %#v. Got %#v.", *expected, got)
	}
	if !reflect.DeepEqual(got.NetworkSettings, expected.NetworkSettings) {
		t.Errorf("InspectContainer: wrong value. Want %#v. Got %#v.", *expected, got)
	}
	got.State.StartedAt = expected.State.StartedAt
	got.State.FinishedAt = expected.State.FinishedAt
	got.Config = expected.Config
	got.Created = expected.Created
	got.NetworkSettings = expected.NetworkSettings
	if !reflect.DeepEqual(got, *expected) {
		t.Errorf("InspectContainer: wrong value. Want %#v. Got %#v.", *expected, got)
	}
}

func TestInspectContainerNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/containers/abc123/json", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("InspectContainer: wrong status code. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func TestStartContainer(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s/start", server.containers[0].ID)
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("StartContainer: wrong status code. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
	if !server.containers[0].State.Running {
		t.Error("StartContainer: did not set the container to running state")
	}
}

func TestStartContainerNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := "/containers/abc123/start"
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("StartContainer: wrong status code. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func TestStartContainerAlreadyRunning(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.containers[0].State.Running = true
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s/start", server.containers[0].ID)
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("StartContainer: wrong status code. Want %d. Got %d.", http.StatusBadRequest, recorder.Code)
	}
}

func TestStopContainer(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.containers[0].State.Running = true
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s/stop", server.containers[0].ID)
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Errorf("StopContainer: wrong status code. Want %d. Got %d.", http.StatusNoContent, recorder.Code)
	}
	if server.containers[0].State.Running {
		t.Error("StopContainer: did not stop the container")
	}
}

func TestStopContainerNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := "/containers/abc123/stop"
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("StopContainer: wrong status code. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func TestStopContainerNotRunning(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s/stop", server.containers[0].ID)
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("StopContainer: wrong status code. Want %d. Got %d.", http.StatusBadRequest, recorder.Code)
	}
}

func TestWaitContainer(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.containers[0].State.Running = true
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s/wait", server.containers[0].ID)
	request, _ := http.NewRequest("POST", path, nil)
	go func() {
		time.Sleep(200e6)
		server.cMut.Lock()
		server.containers[0].State.Running = false
		server.cMut.Unlock()
	}()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("WaitContainer: wrong status. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
	expected := `{"StatusCode":0}`
	if body := recorder.Body.String(); body != expected {
		t.Errorf("WaitContainer: wrong body. Want %q. Got %q.", expected, body)
	}
}

func TestWaitContainerNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := "/containers/abc123/wait"
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("WaitContainer: wrong status code. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func TestAttachContainer(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.containers[0].State.Running = true
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s/attach?logs=1", server.containers[0].ID)
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	lines := []string{
		fmt.Sprintf("\x01\x00\x00\x00\x03\x00\x00\x00Container %q is running", server.containers[0].ID),
		"What happened?",
		"Something happened",
	}
	expected := strings.Join(lines, "\n") + "\n"
	if body := recorder.Body.String(); body == expected {
		t.Errorf("AttachContainer: wrong body. Want %q. Got %q.", expected, body)
	}
}

func TestAttachContainerNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := "/containers/abc123/attach?logs=1"
	request, _ := http.NewRequest("POST", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("AttachContainer: wrong status. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func TestRemoveContainer(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s", server.containers[0].ID)
	request, _ := http.NewRequest("DELETE", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Errorf("RemoveContainer: wrong status. Want %d. Got %d.", http.StatusNoContent, recorder.Code)
	}
	if len(server.containers) > 0 {
		t.Error("RemoveContainer: did not remove the container.")
	}
}

func TestRemoveContainerNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/abc123")
	request, _ := http.NewRequest("DELETE", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("RemoveContainer: wrong status. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func TestRemoveContainerRunning(t *testing.T) {
	server := DockerServer{}
	addContainers(&server, 1)
	server.containers[0].State.Running = true
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/containers/%s", server.containers[0].ID)
	request, _ := http.NewRequest("DELETE", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("RemoveContainer: wrong status. Want %d. Got %d.", http.StatusInternalServerError, recorder.Code)
	}
	if len(server.containers) < 1 {
		t.Error("RemoveContainer: should not remove the container.")
	}
}

func TestPullImage(t *testing.T) {
	server := DockerServer{imgIDs: make(map[string]string)}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/images/create?fromImage=base", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("PullImage: wrong status. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
	if len(server.images) != 1 {
		t.Errorf("PullImage: Want 1 image. Got %d.", len(server.images))
	}
	if _, ok := server.imgIDs["base"]; !ok {
		t.Error("PullImage: Repository should not be empty.")
	}
}

func TestPushImage(t *testing.T) {
	server := DockerServer{imgIDs: map[string]string{"tsuru/python": "a123"}}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/images/tsuru/python/push", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("PushImage: wrong status. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
}

func TestPushImageNotFound(t *testing.T) {
	server := DockerServer{}
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("POST", "/images/tsuru/python/push", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("PushImage: wrong status. Want %d. Got %d.", http.StatusNotFound, recorder.Code)
	}
}

func addContainers(server *DockerServer, n int) {
	server.cMut.Lock()
	defer server.cMut.Unlock()
	for i := 0; i < n; i++ {
		date := time.Now().Add(time.Duration((rand.Int() % (i + 1))) * time.Hour)
		container := docker.Container{
			ID:      fmt.Sprintf("%x", rand.Int()%10000),
			Created: date,
			Path:    "ls",
			Args:    []string{"-la", ".."},
			Config: &docker.Config{
				Hostname:     fmt.Sprintf("docker-%d", i),
				AttachStdout: true,
				AttachStderr: true,
				Env:          []string{"ME=you", fmt.Sprintf("NUMBER=%d", i)},
				Cmd:          []string{"ls", "-la", ".."},
				Dns:          nil,
				Image:        "base",
			},
			State: docker.State{
				Running:   false,
				Pid:       400 + i,
				ExitCode:  0,
				StartedAt: date,
			},
			Image: "b750fe79269d2ec9a3c593ef05b4332b1d1a02a62b4accb2c21d589ff2f5f2dc",
			NetworkSettings: &docker.NetworkSettings{
				IPAddress:   fmt.Sprintf("10.10.10.%d", i+2),
				IPPrefixLen: 24,
				Gateway:     "10.10.10.1",
				Bridge:      "docker0",
				PortMapping: map[string]docker.PortMapping{
					"Tcp": {"8888": fmt.Sprintf("%d", 49600+i)},
				},
			},
			ResolvConfPath: "/etc/resolv.conf",
		}
		server.containers = append(server.containers, &container)
	}
}

func addImages(server *DockerServer, n int, repo bool) {
	server.iMut.Lock()
	defer server.iMut.Unlock()
	if server.imgIDs == nil {
		server.imgIDs = make(map[string]string)
	}
	for i := 0; i < n; i++ {
		date := time.Now().Add(time.Duration((rand.Int() % (i + 1))) * time.Hour)
		image := docker.Image{
			ID:      fmt.Sprintf("%x", rand.Int()%10000),
			Created: date,
		}
		server.images = append(server.images, image)
		if repo {
			repo := "docker/python-" + image.ID
			server.imgIDs[repo] = image.ID
		}
	}
}

func TestListImages(t *testing.T) {
	server := DockerServer{}
	addImages(&server, 2, false)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/images/json?all=1", nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("ListImages: wrong status. Want %d. Got %d.", http.StatusOK, recorder.Code)
	}
	expected := make([]docker.APIImages, 2)
	for i, image := range server.images {
		expected[i] = docker.APIImages{
			ID:      image.ID,
			Created: image.Created.Unix(),
		}
	}
	var got []docker.APIImages
	err := json.NewDecoder(recorder.Body).Decode(&got)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ListImages. Want %#v. Got %#v.", expected, got)
	}
}

func TestRemoveImage(t *testing.T) {
	server := DockerServer{}
	addImages(&server, 1, false)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := fmt.Sprintf("/images/%s", server.images[0].ID)
	request, _ := http.NewRequest("DELETE", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Errorf("RemoveImage: wrong status. Want %d. Got %d.", http.StatusNoContent, recorder.Code)
	}
	if len(server.images) > 0 {
		t.Error("RemoveImage: did not remove the image.")
	}
}

func TestRemoveImageByName(t *testing.T) {
	server := DockerServer{}
	addImages(&server, 1, true)
	server.buildMuxer()
	recorder := httptest.NewRecorder()
	path := "/images/docker/python-" + server.images[0].ID
	request, _ := http.NewRequest("DELETE", path, nil)
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Errorf("RemoveImage: wrong status. Want %d. Got %d.", http.StatusNoContent, recorder.Code)
	}
	if len(server.images) > 0 {
		t.Error("RemoveImage: did not remove the image.")
	}
}
