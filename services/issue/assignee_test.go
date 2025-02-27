// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func TestDeleteNotPassedAssignee(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Fake issue with assignees
	issue, err := issues_model.GetIssueWithAttrsByID(1)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, len(issue.Assignees))

	user1, err := user_model.GetUserByID(1) // This user is already assigned (see the definition in fixtures), so running  UpdateAssignee should unassign him
	assert.NoError(t, err)

	// Check if he got removed
	isAssigned, err := issues_model.IsUserAssignedToIssue(db.DefaultContext, issue, user1)
	assert.NoError(t, err)
	assert.True(t, isAssigned)

	// Clean everyone
	err = DeleteNotPassedAssignee(issue, user1, []*user_model.User{})
	assert.NoError(t, err)
	assert.EqualValues(t, 0, len(issue.Assignees))

	// Check they're gone
	assert.NoError(t, issue.LoadAssignees(db.DefaultContext))
	assert.EqualValues(t, 0, len(issue.Assignees))
	assert.Empty(t, issue.Assignee)
}
