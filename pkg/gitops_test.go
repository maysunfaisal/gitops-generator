//
// Copyright 2021-2022 Red Hat, Inc.
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

package gitops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gitopsv1alpha1 "github.com/redhat-developer/gitops-generator/api/v1alpha1"
	"github.com/redhat-developer/gitops-generator/pkg/testutils"
	"github.com/redhat-developer/gitops-generator/pkg/util/ioutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCloneGenerateAndPush(t *testing.T) {
	repo := "git@github.com:testing/testing.git"
	outputPath := "/fake/path"
	repoPath := "/fake/path/test-component"
	componentName := "test-component"
	component := gitopsv1alpha1.Component{
		Spec: gitopsv1alpha1.ComponentSpec{
			ContainerImage: "testimage:latest",
			Source: gitopsv1alpha1.ComponentSource{
				ComponentSourceUnion: gitopsv1alpha1.ComponentSourceUnion{
					GitSource: &gitopsv1alpha1.GitSource{
						URL: repo,
					},
				},
			},
			TargetPort: 5000,
		},
	}
	component.Name = "test-component"
	fs := ioutils.NewMemoryFilesystem()
	readOnlyFs := ioutils.NewReadOnlyFs()

	tests := []struct {
		name          string
		fs            afero.Afero
		component     gitopsv1alpha1.Component
		errors        *testutils.ErrorStack
		outputs       [][]byte
		want          []testutils.Execution
		wantErrString string
	}{
		{
			name:      "No errors",
			fs:        fs,
			component: component,
			errors:    &testutils.ErrorStack{},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", filepath.Join("components", componentName, "base")},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate GitOps base resources for component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
		},
		{
			name:      "Git clone failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
			},
			wantErrString: "test error",
		},
		{
			name:      "Git switch failure, git checkout failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission denied"),
					errors.New("Fatal error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
			},
			wantErrString: "failed to checkout branch \"main\" in \"/fake/path/test-component\" \"test output1\": Permission denied",
		},
		{
			name:      "Git switch failure, git checkout success",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
				[]byte("test output8"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", filepath.Join("components", componentName, "base")},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate GitOps base resources for component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "",
		},
		{
			name:      "rm -rf failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
			},
			wantErrString: "failed to delete \"components/test-component/base\" folder in repository in \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git add failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
			},
			wantErrString: "failed to add files for component \"test-component\" to repository in \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git diff failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
			},
			wantErrString: "failed to check git diff in repository \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git commit failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate GitOps base resources for component %s", componentName)},
				},
			},
			wantErrString: "failed to commit files to repository in \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git push failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate GitOps base resources for component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "failed push remote to repository \"git@github.com:testing/testing.git\" \"test output1\": Fatal error",
		},
		{
			name:      "gitops generate failure",
			fs:        readOnlyFs,
			component: component,
			errors:    &testutils.ErrorStack{},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
			},
			wantErrString: "failed to generate the gitops resources in \"/fake/path/test-component/components/test-component/base\" for component \"test-component\"",
		},
		{
			name: "gitops generate failure - image component",
			fs:   readOnlyFs,
			component: gitopsv1alpha1.Component{
				ObjectMeta: v1.ObjectMeta{
					Name: "test-component",
				},
				Spec: gitopsv1alpha1.ComponentSpec{
					ContainerImage: "quay.io/test/test",
					TargetPort:     5000,
				},
			},
			errors: &testutils.ErrorStack{},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, "test-component"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", "components/test-component/base"},
				},
			},
			wantErrString: "failed to generate the gitops resources in \"/fake/path/test-component/components/test-component/base\" for component \"test-component\": failed to MkDirAll",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := testutils.NewMockExecutor(tt.outputs...)
			e.Errors = tt.errors
			err := CloneGenerateAndPush(outputPath, repo, tt.component, e, tt.fs, "main", "/", true)

			if tt.wantErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantErrString, err)
			} else {
				testutils.AssertNoError(t, err)
			}

			assert.Equal(t, tt.want, e.Executed, "command executed should be equal")
		})
	}
}

func TestGenerateOverlaysAndPush(t *testing.T) {
	repo := "git@github.com:testing/testing.git"
	outputPath := "/fake/path"
	repoPath := "/fake/path/test-application"
	componentName := "test-component"
	applicationName := "test-application"
	environmentName := "environment"
	imageName := "image"
	namespace := "namespace"
	component := gitopsv1alpha1.BindingComponentConfiguration{
		Name:     componentName,
		Replicas: 2,
	}
	component.Name = "test-component"
	fs := ioutils.NewMemoryFilesystem()
	readOnlyFs := ioutils.NewReadOnlyFs()

	tests := []struct {
		name            string
		fs              afero.Afero
		component       gitopsv1alpha1.BindingComponentConfiguration
		environment     gitopsv1alpha1.Environment
		errors          *testutils.ErrorStack
		outputs         [][]byte
		applicationName string
		environmentName string
		imageName       string
		namespace       string
		want            []testutils.Execution
		wantErrString   string
	}{
		{
			name:      "No errors",
			fs:        fs,
			component: component,
			errors:    &testutils.ErrorStack{},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate %s environment overlays for component %s", environmentName, componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
		},
		{
			name:      "Git clone failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
			},
			wantErrString: "test error",
		},
		{
			name:      "Git switch failure, git checkout failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission denied"),
					errors.New("Fatal error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
			},
			wantErrString: "failed to checkout branch \"main\" in \"/fake/path/test-application\" \"test output1\": Permission denied",
		},
		{
			name:      "Git switch failure, git checkout success",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate %s environment overlays for component %s", environmentName, componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "",
		},
		{
			name:      "git add failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
			},
			wantErrString: "failed to add files for component \"test-component\" to repository in \"/fake/path/test-application\" \"test output1\": Fatal error",
		},
		{
			name:      "git diff failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
			},
			wantErrString: "failed to check git diff in repository \"/fake/path/test-application\" \"test output1\": Permission Denied",
		},
		{
			name:      "git commit failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate %s environment overlays for component %s", environmentName, componentName)},
				},
			},
			wantErrString: "failed to commit files to repository in \"/fake/path/test-application\" \"test output1\": Fatal error",
		},
		{
			name:      "git push failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Generate %s environment overlays for component %s", environmentName, componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "failed push remote to repository \"git@github.com:testing/testing.git\" \"test output1\": Fatal error",
		},
		{
			name:            "gitops generate failure",
			fs:              readOnlyFs,
			component:       component,
			errors:          &testutils.ErrorStack{},
			applicationName: applicationName,
			environmentName: environmentName,
			imageName:       imageName,
			namespace:       namespace,
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, applicationName},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
			},
			wantErrString: "failed to generate the gitops resources in overlays dir \"/fake/path/test-application/components/test-component/overlays/environment\" for component \"test-component\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := testutils.NewMockExecutor(tt.outputs...)
			e.Errors = tt.errors
			generatedResources := make(map[string][]string)
			err := GenerateOverlaysAndPush(outputPath, true, repo, tt.component, tt.environment, tt.applicationName, tt.environmentName, tt.imageName, tt.namespace, e, tt.fs, "main", "/", true, generatedResources)

			if tt.wantErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantErrString, err)
			} else {
				testutils.AssertNoError(t, err)
				assert.Equal(t, 1, len(generatedResources), "should be equal")
				hasGitopsGeneratedResource := map[string]bool{
					"deployment-patch.yaml": true,
				}

				for _, generatedRes := range generatedResources[componentName] {
					assert.True(t, hasGitopsGeneratedResource[generatedRes], "should be equal")
				}
			}

			assert.Equal(t, tt.want, e.Executed, "command executed should be equal")
		})
	}
}

func TestRemoveAndPush(t *testing.T) {
	repo := "git@github.com:testing/testing.git"
	outputPath := "/fake/path"
	repoPath := "/fake/path/test-component"
	componentPath := "/fake/path/test-component/components/test-component"
	componentBasePath := "/fake/path/test-component/components/test-component/base"
	componentName := "test-component"
	component := gitopsv1alpha1.Component{
		Spec: gitopsv1alpha1.ComponentSpec{
			Source: gitopsv1alpha1.ComponentSource{
				ComponentSourceUnion: gitopsv1alpha1.ComponentSourceUnion{
					GitSource: &gitopsv1alpha1.GitSource{
						URL: repo,
					},
				},
			},
			TargetPort: 5000,
		},
	}
	component.Name = "test-component"
	fs := ioutils.NewMemoryFilesystem()
	readOnlyFs := ioutils.NewReadOnlyFs()

	tests := []struct {
		name          string
		fs            afero.Afero
		component     gitopsv1alpha1.Component
		errors        *testutils.ErrorStack
		outputs       [][]byte
		want          []testutils.Execution
		wantErrString string
	}{
		{
			name:      "No errors",
			fs:        fs,
			component: component,
			errors:    &testutils.ErrorStack{},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
		},
		{
			name:      "Git clone failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
			},
			wantErrString: "test error",
		},
		{
			name:      "Git switch failure, git checkout failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission denied"),
					errors.New("Fatal error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
			},
			wantErrString: "failed to checkout branch \"main\" in \"/fake/path/test-component\" \"test output1\": Permission denied",
		},
		{
			name:      "Git switch failure, git checkout success",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					nil,
					errors.New("test error"),
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
				[]byte("test output8"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"checkout", "-b", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "",
		},
		{
			name:      "rm -rf failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
			},
			wantErrString: "failed to delete \"/fake/path/test-component/components/test-component\" folder in repository in \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git add failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
			},
			wantErrString: "failed to add files for component \"test-component\" to repository in \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git diff failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Permission Denied"),
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
			},
			wantErrString: "failed to check git diff in repository \"/fake/path/test-component\" \"test output1\": Permission Denied",
		},
		{
			name:      "git commit failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
			},
			wantErrString: "failed to commit files to repository in \"/fake/path/test-component\" \"test output1\": Fatal error",
		},
		{
			name:      "git push failure",
			fs:        fs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("Fatal error"),
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
				},
			},
			outputs: [][]byte{
				[]byte("test output1"),
				[]byte("test output2"),
				[]byte("test output3"),
				[]byte("test output4"),
				[]byte("test output5"),
				[]byte("test output6"),
				[]byte("test output7"),
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"add", "."},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"--no-pager", "diff", "--cached"},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"commit", "-m", fmt.Sprintf("Removed component %s", componentName)},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"push", "origin", "main"},
				},
			},
			wantErrString: "failed push remote to repository \"git@github.com:testing/testing.git\" \"test output1\": Fatal error",
		},
		{
			name:      "kustomize generate failure",
			fs:        readOnlyFs,
			component: component,
			errors: &testutils.ErrorStack{
				Errors: []error{
					errors.New("access error"),
					nil,
					nil,
					nil,
				},
			},
			want: []testutils.Execution{
				{
					BaseDir: outputPath,
					Command: "git",
					Args:    []string{"clone", repo, component.Name},
				},
				{
					BaseDir: repoPath,
					Command: "git",
					Args:    []string{"switch", "main"},
				},
				{
					BaseDir: repoPath,
					Command: "rm",
					Args:    []string{"-rf", componentPath},
				},
			},
			wantErrString: "failed to re-generate the gitops resources in \"/fake/path/test-component/components/test-component\" for component \"test-component\": access error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := testutils.NewMockExecutor(tt.outputs...)
			e.Errors = tt.errors

			if err := Generate(fs, repoPath, componentBasePath, tt.component); err != nil {
				t.Errorf("unexpected error %v", err)
				return
			}

			err := RemoveAndPush(outputPath, repo, tt.component.Name, e, tt.fs, "main", "/", true)

			if tt.wantErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantErrString, err)
			} else {
				testutils.AssertNoError(t, err)
			}

			assert.Equal(t, tt.want, e.Executed, "command executed should be equal")
		})
	}
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		outputPath string
		args       string
		wantErr    bool
	}{
		{
			name:    "Simple command to execute",
			command: "git",
			args:    "help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new executor
			e := NewCmdExecutor()
			_, err := e.Execute(tt.outputPath, tt.command, tt.args)
			if !tt.wantErr && (err != nil) {
				t.Errorf("TestExecute() unexpected error value: %v", err)
			}
		})
	}
}

func TestGenerateAndPush(t *testing.T) {
	repo := "http://github.com/testing/testing.git"
	outputPath := "/fake/path"
	component := gitopsv1alpha1.Component{
		Spec: gitopsv1alpha1.ComponentSpec{
			ContainerImage: "testimage:latest",
			Source: gitopsv1alpha1.ComponentSource{
				ComponentSourceUnion: gitopsv1alpha1.ComponentSourceUnion{
					GitSource: &gitopsv1alpha1.GitSource{
						URL: repo,
					},
				},
			},
			TargetPort: 5000,
		},
	}
	component.Name = "test-component"
	component.Spec.ComponentName = "test-component"
	fs := ioutils.NewMemoryFilesystem()

	tests := []struct {
		name          string
		fs            afero.Afero
		component     gitopsv1alpha1.Component
		errors        *testutils.ErrorStack
		outputs       [][]byte
		want          []testutils.Execution
		wantErrString string
	}{
		{
			name:      "No errors. GenerateAndPush test with no push",
			fs:        fs,
			component: component,
			errors:    &testutils.ErrorStack{},
			want:      []testutils.Execution{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := testutils.NewMockExecutor(tt.outputs...)
			e.Errors = tt.errors
			err := GenerateAndPush(outputPath, repo, tt.component, e, tt.fs, "main", false, "KAM CLI", nil)

			if tt.wantErrString != "" {
				testutils.AssertErrorMatch(t, tt.wantErrString, err)
			} else {
				testutils.AssertNoError(t, err)
			}

			assert.Equal(t, tt.want, e.Executed, "command executed should be equal")
		})
	}
}

func TestGetCommitIDFromRepo(t *testing.T) {
	// Create an empty git repository and git commit to test with
	fs := ioutils.NewFilesystem()
	tempDir, err := fs.TempDir(os.TempDir(), "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = createEmptyGitRepository(NewCmdExecutor(), tempDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	commitID, err := getCommitIDFromDotGit(tempDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	e := NewCmdExecutor()
	tests := []struct {
		name     string
		e        Executor
		repoPath string
		want     string
		wantErr  bool
	}{
		{
			name:     "No errors, successfully retrieve git commit ID",
			e:        e,
			repoPath: tempDir,
			want:     commitID,
			wantErr:  false,
		},
		{
			name:     "Invalid git repo, no commit ID",
			e:        e,
			repoPath: os.TempDir(),
			want:     "",
			wantErr:  true,
		},
		{
			name:     "Test with mock executor, should pass",
			e:        testutils.NewMockExecutor(),
			repoPath: os.TempDir(),
			want:     "ca82a6dff817ec66f44342007202690a93763949",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commitID, err := GetCommitIDFromRepo(fs, tt.e, tt.repoPath)

			if err != nil && !tt.wantErr {
				t.Errorf("TestGetCommitIDFromRepo() unexpected error: %s", err.Error())
			}
			if err == nil && tt.wantErr {
				t.Errorf("TestGetCommitIDFromRepo() did not get expected error")
			}
			if commitID != tt.want {
				t.Errorf("TestGetCommitIDFromRepo() wanted: %v, got: %v", tt.want, commitID)
			}
		})
	}
}

// createEmptyGitRepository generates an empty git repository under the specified folder
func createEmptyGitRepository(e Executor, repoPath string) error {
	// Initialize the Git repository
	if out, err := e.Execute(repoPath, "git", "init"); err != nil {
		return fmt.Errorf("Unable to intialize git repository in %q %q: %s", repoPath, out, err)
	}

	// Create an empty commit
	if out, err := e.Execute(repoPath, "git", "-c", "user.name='Test User'", "-c", "user.email='test@test.org'", "commit", "--allow-empty", "-m", "\"Empty commit\""); err != nil {
		return fmt.Errorf("Unable to create empty commit in %q %q: %s", repoPath, out, err)
	}
	return nil
}

// getCommitIDFromDotGit returns the latest commit ID for the default branch in the given git repository
func getCommitIDFromDotGit(repoPath string) (string, error) {
	fs := ioutils.NewFilesystem()
	var fileBytes []byte
	fileBytes, err := fs.ReadFile(filepath.Join(repoPath, ".git", "refs", "heads", "master"))
	if err != nil {
		return "", err
	}
	return string(fileBytes), nil
}
