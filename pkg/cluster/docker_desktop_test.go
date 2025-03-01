package cluster

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoOp(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	settings, err := f.d4m.settings(ctx)
	require.NoError(t, err)

	err = f.d4m.writeSettings(ctx, settings)
	require.NoError(t, err)

	assert.Equal(t,
		f.postSettings,
		f.readerToMap(strings.NewReader(postSettingsJSON)))
}

func TestEnableKubernetes(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	f.settings = getSettingsOldJSON

	ctx := context.Background()
	settings, err := f.d4m.settings(ctx)
	require.NoError(t, err)

	changed, err := f.d4m.setK8sEnabled(settings, true)
	assert.True(t, changed)
	require.NoError(t, err)

	err = f.d4m.writeSettings(ctx, settings)
	require.NoError(t, err)

	expected := strings.Replace(postSettingsJSON,
		`"kubernetes":{"enabled":false,`,
		`"kubernetes":{"enabled":true,`, 1)
	assert.Equal(t,
		f.postSettings,
		f.readerToMap(strings.NewReader(expected)))
}

func TestEnableKubernetesOldFormat(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	settings, err := f.d4m.settings(ctx)
	require.NoError(t, err)

	changed, err := f.d4m.setK8sEnabled(settings, true)
	assert.True(t, changed)
	require.NoError(t, err)

	err = f.d4m.writeSettings(ctx, settings)
	require.NoError(t, err)

	expected := strings.Replace(postSettingsJSON,
		`"kubernetes":{"enabled":false,`,
		`"kubernetes":{"enabled":true,`, 1)
	assert.Equal(t,
		f.postSettings,
		f.readerToMap(strings.NewReader(expected)))
}

func TestMinCPUs(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	settings, err := f.d4m.settings(ctx)
	require.NoError(t, err)

	changed, err := f.d4m.ensureMinCPU(settings, 4)
	assert.True(t, changed)
	require.NoError(t, err)

	err = f.d4m.writeSettings(ctx, settings)
	require.NoError(t, err)

	expected := strings.Replace(postSettingsJSON,
		`"cpus":2`,
		`"cpus":4`, 1)
	assert.Equal(t,
		f.postSettings,
		f.readerToMap(strings.NewReader(expected)))
}

func TestMaxCPUs(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	settings, err := f.d4m.settings(ctx)
	require.NoError(t, err)

	changed, err := f.d4m.ensureMinCPU(settings, 40)
	assert.False(t, changed)
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), "desired cpus (40) greater than max allowed (8)")
	}
}

func TestLookupMap(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	settings, err := f.d4m.settings(ctx)
	require.NoError(t, err)

	_, err = f.d4m.lookupMapAt(settings, "vm.kubernetes.honk")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), `nothing found at DockerDesktop setting "vm.kubernetes.honk"`)
	}
}

func TestSetSettingValueInvalidKey(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	err := f.d4m.SetSettingValue(ctx, "vm.doesNotExist", "4")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), `nothing found at DockerDesktop setting "vm.doesNotExist"`)
	}
}

func TestSetSettingValueInvalidSet(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	err := f.d4m.SetSettingValue(ctx, "vm.resources.cpus.value.doesNotExist", "4")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), `expected map at DockerDesktop setting "vm.resources.cpus.value", got: float64`)
	}
}

func TestSetSettingValueFloat(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	err := f.d4m.SetSettingValue(ctx, "vm.resources.cpus", "4")
	require.NoError(t, err)

	expected := strings.Replace(postSettingsJSON,
		`"cpus":2`,
		`"cpus":4`, 1)
	assert.Equal(t,
		f.postSettings,
		f.readerToMap(strings.NewReader(expected)))

	f.postSettings = nil
	err = f.d4m.SetSettingValue(ctx, "vm.resources.cpus", "2")
	require.NoError(t, err)
	assert.Nil(t, f.postSettings)
}

func TestSetSettingValueFloatLimit(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	err := f.d4m.SetSettingValue(ctx, "vm.resources.cpus", "100")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), `setting value "vm.resources.cpus": 100 greater than max allowed`)
	}
	err = f.d4m.SetSettingValue(ctx, "vm.resources.cpus", "0")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), `setting value "vm.resources.cpus": 0 less than min allowed`)
	}
}

func TestSetSettingValueBool(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	err := f.d4m.SetSettingValue(ctx, "vm.kubernetes.enabled", "true")
	require.NoError(t, err)

	expected := strings.Replace(postSettingsJSON,
		`"enabled":false`,
		`"enabled":true`, 1)
	assert.Equal(t,
		f.postSettings,
		f.readerToMap(strings.NewReader(expected)))

	f.postSettings = nil
	err = f.d4m.SetSettingValue(ctx, "vm.kubernetes.enabled", "false")
	require.NoError(t, err)
	assert.Nil(t, f.postSettings)
}

func TestSetSettingValueFileSharing(t *testing.T) {
	f := newD4MFixture(t)
	defer f.TearDown()

	ctx := context.Background()
	err := f.d4m.SetSettingValue(ctx, "vm.fileSharing", "/x,/y")
	require.NoError(t, err)

	expected := strings.Replace(postSettingsJSON,
		`[{"path":"/Users","cached":false},{"path":"/Volumes","cached":false},{"path":"/private","cached":false},{"path":"/tmp","cached":false}]`,
		`[{"path":"/x","cached":false},{"path":"/y","cached":false}]`, 1)
	assert.Equal(t,
		f.postSettings,
		f.readerToMap(strings.NewReader(expected)))

}

var getSettingsOldJSON = `{"vm":{"proxy":{"exclude":{"value":"","locked":false},"http":{"value":"","locked":false},"https":{"value":"","locked":false},"mode":{"value":"system","locked":false}},"daemon":{"locks":[],"json":"{\"debug\":true,\"experimental\":false}"},"resources":{"cpus":{"value":2,"min":1,"locked":false,"max":8},"memoryMiB":{"value":8192,"min":1024,"locked":false,"max":16384},"diskSizeMiB":{"value":61035,"used":18486,"locked":false},"dataFolder":{"value":"\/Users\/nick\/Library\/Containers\/com.docker.docker\/Data\/vms\/0\/data","locked":false},"swapMiB":{"value":1024,"min":512,"locked":false,"max":4096}},"fileSharing":{"value":[{"path":"\/Users","cached":false},{"path":"\/Volumes","cached":false},{"path":"\/private","cached":false},{"path":"\/tmp","cached":false}],"locked":false},"kubernetes":{"enabled":{"value":false,"locked":false},"stackOrchestrator":{"value":false,"locked":false},"showSystemContainers":{"value":false,"locked":false}},"network":{"dns":{"locked":false},"vpnkitCIDR":{"value":"192.168.65.0\/24","locked":false},"automaticDNS":{"value":true,"locked":false}}},"desktop":{"exportInsecureDaemon":{"value":false,"locked":false},"useGrpcfuse":{"value":true,"locked":false},"backupData":{"value":false,"locked":false},"checkForUpdates":{"value":true,"locked":false},"useCredentialHelper":{"value":true,"locked":false},"autoStart":{"value":false,"locked":false},"analyticsEnabled":{"value":true,"locked":false}},"wslIntegration":{"distros":{"value":[],"locked":false},"enableIntegrationWithDefaultWslDistro":{"value":false,"locked":false}},"cli":{"useCloudCli":{"value":true,"locked":false},"experimental":{"value":true,"locked":false}}}`

var getSettingsJSON = `{"vm":{"proxy":{"exclude":"","http":"","https":"","mode":"system"},"daemon":{"locks":[],"json":"{\"debug\":true,\"experimental\":false}"},"resources":{"cpus":{"value":2,"min":1,"locked":false,"max":8},"memoryMiB":{"value":8192,"min":1024,"locked":false,"max":16384},"diskSizeMiB":{"value":61035,"used":18486,"locked":false},"dataFolder":"\/Users\/nick\/Library\/Containers\/com.docker.docker\/Data\/vms\/0\/data","swapMiB":{"value":1024,"min":512,"locked":false,"max":4096}},"fileSharing":{"value":[{"path":"\/Users","cached":false},{"path":"\/Volumes","cached":false},{"path":"\/private","cached":false},{"path":"\/tmp","cached":false}],"locked":false},"kubernetes":{"enabled":false,"stackOrchestrator":false,"showSystemContainers":false},"network":{"dns":{"locked":false},"vpnkitCIDR":"192.168.65.0\/24","automaticDNS":true}},"desktop":{"exportInsecureDaemon":false,"useGrpcfuse":true,"backupData":false,"checkForUpdates":true,"useCredentialHelper":true,"autoStart":false,"analyticsEnabled":true},"wslIntegration":{"distros":[],"enableIntegrationWithDefaultWslDistro":false},"cli":{"useCloudCli":true,"experimental":true}}`

var postSettingsJSON = `{"desktop":{"exportInsecureDaemon":false,"useGrpcfuse":true,"backupData":false,"checkForUpdates":true,"useCredentialHelper":true,"autoStart":false,"analyticsEnabled":true},"cli":{"useCloudCli":true,"experimental":true},"vm":{"daemon":"{\"debug\":true,\"experimental\":false}","fileSharing":[{"path":"/Users","cached":false},{"path":"/Volumes","cached":false},{"path":"/private","cached":false},{"path":"/tmp","cached":false}],"kubernetes":{"enabled":false,"stackOrchestrator":false,"showSystemContainers":false},"network":{"vpnkitCIDR":"192.168.65.0/24","automaticDNS":true},"proxy":{"exclude":"","http":"","https":"","mode":"system"},"resources":{"cpus":2,"memoryMiB":8192,"diskSizeMiB":61035,"dataFolder":"/Users/nick/Library/Containers/com.docker.docker/Data/vms/0/data","swapMiB":1024}},"wslIntegration":{"distros":[],"enableIntegrationWithDefaultWslDistro":false}}`

type d4mFixture struct {
	t            *testing.T
	d4m          *DockerDesktopClient
	settings     string
	postSettings map[string]interface{}
}

func newD4MFixture(t *testing.T) *d4mFixture {
	f := &d4mFixture{t: t}
	f.settings = getSettingsJSON
	f.d4m = &DockerDesktopClient{guiClient: f}
	return f
}

func (f *d4mFixture) readerToMap(r io.Reader) map[string]interface{} {
	result := make(map[string]interface{})
	err := json.NewDecoder(r).Decode(&result)
	require.NoError(f.t, err)
	return result
}

func (f *d4mFixture) Do(r *http.Request) (*http.Response, error) {
	require.Equal(f.t, r.URL.Path, "/settings")
	if r.Method == "POST" {
		f.postSettings = f.readerToMap(r.Body)

		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       closeReader{strings.NewReader("")},
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       closeReader{strings.NewReader(f.settings)},
	}, nil
}

func (f *d4mFixture) TearDown() {
}

type closeReader struct {
	io.Reader
}

func (c closeReader) Close() error { return nil }
