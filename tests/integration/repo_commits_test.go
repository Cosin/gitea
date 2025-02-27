// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestRepoCommits(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	// Request repository commits page
	req := NewRequest(t, "GET", "/user2/repo1/commits/branch/master")
	resp := session.MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	commitURL, exists := doc.doc.Find("#commits-table tbody tr td.sha a").Attr("href")
	assert.True(t, exists)
	assert.NotEmpty(t, commitURL)
}

func doTestRepoCommitWithStatus(t *testing.T, state string, classes ...string) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	// Request repository commits page
	req := NewRequest(t, "GET", "/user2/repo1/commits/branch/master")
	resp := session.MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	// Get first commit URL
	commitURL, exists := doc.doc.Find("#commits-table tbody tr td.sha a").Attr("href")
	assert.True(t, exists)
	assert.NotEmpty(t, commitURL)

	// Call API to add status for commit
	t.Run("CreateStatus", doAPICreateCommitStatus(NewAPITestContext(t, "user2", "repo1"), path.Base(commitURL), api.CommitStatusState(state)))

	req = NewRequest(t, "GET", "/user2/repo1/commits/branch/master")
	resp = session.MakeRequest(t, req, http.StatusOK)

	doc = NewHTMLParser(t, resp.Body)
	// Check if commit status is displayed in message column
	sel := doc.doc.Find("#commits-table tbody tr td.message a.commit-statuses-trigger .commit-status")
	assert.Equal(t, 1, sel.Length())
	for _, class := range classes {
		assert.True(t, sel.HasClass(class))
	}

	// By SHA
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/commits/"+path.Base(commitURL)+"/statuses")
	reqOne := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/commits/"+path.Base(commitURL)+"/status")
	testRepoCommitsWithStatus(t, session.MakeRequest(t, req, http.StatusOK), session.MakeRequest(t, reqOne, http.StatusOK), state)

	// By Ref
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/commits/master/statuses")
	reqOne = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/commits/master/status")
	testRepoCommitsWithStatus(t, session.MakeRequest(t, req, http.StatusOK), session.MakeRequest(t, reqOne, http.StatusOK), state)
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/commits/v1.1/statuses")
	reqOne = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/commits/v1.1/status")
	testRepoCommitsWithStatus(t, session.MakeRequest(t, req, http.StatusOK), session.MakeRequest(t, reqOne, http.StatusOK), state)
}

func testRepoCommitsWithStatus(t *testing.T, resp, respOne *httptest.ResponseRecorder, state string) {
	var statuses []*api.CommitStatus
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &statuses))
	var status api.CombinedStatus
	assert.NoError(t, json.Unmarshal(respOne.Body.Bytes(), &status))
	assert.NotNil(t, status)

	if assert.Len(t, statuses, 1) {
		assert.Equal(t, api.CommitStatusState(state), statuses[0].State)
		assert.Equal(t, setting.AppURL+"api/v1/repos/user2/repo1/statuses/65f1bf27bc3bf70f64657658635e66094edbcb4d", statuses[0].URL)
		assert.Equal(t, "http://test.ci/", statuses[0].TargetURL)
		assert.Equal(t, "", statuses[0].Description)
		assert.Equal(t, "testci", statuses[0].Context)

		assert.Len(t, status.Statuses, 1)
		assert.Equal(t, statuses[0], status.Statuses[0])
		assert.Equal(t, "65f1bf27bc3bf70f64657658635e66094edbcb4d", status.SHA)
	}
}

func TestRepoCommitsWithStatusPending(t *testing.T) {
	doTestRepoCommitWithStatus(t, "pending", "octicon-dot-fill", "yellow")
}

func TestRepoCommitsWithStatusSuccess(t *testing.T) {
	doTestRepoCommitWithStatus(t, "success", "octicon-check", "green")
}

func TestRepoCommitsWithStatusError(t *testing.T) {
	doTestRepoCommitWithStatus(t, "error", "gitea-exclamation", "red")
}

func TestRepoCommitsWithStatusFailure(t *testing.T) {
	doTestRepoCommitWithStatus(t, "failure", "octicon-x", "red")
}

func TestRepoCommitsWithStatusWarning(t *testing.T) {
	doTestRepoCommitWithStatus(t, "warning", "gitea-exclamation", "yellow")
}
