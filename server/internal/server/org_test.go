package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/llmariner/rbac-manager/pkg/auth"
	v1 "github.com/llmariner/user-manager/api/v1"
	"github.com/llmariner/user-manager/server/internal/config"
	"github.com/llmariner/user-manager/server/internal/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOrganization(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	isrv := NewInternal(st)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		title := fmt.Sprintf("test %d", i)
		cresp, err := srv.CreateOrganization(ctx, &v1.CreateOrganizationRequest{
			Title: title,
		})
		assert.NoError(t, err)
		assert.Equal(t, title, cresp.Title)

		// Delete the default user to make the rest of the test simple.
		_, err = srv.DeleteOrganizationUser(ctx, &v1.DeleteOrganizationUserRequest{
			OrganizationId: cresp.Id,
			UserId:         defaultUserID,
		})
		assert.NoError(t, err)

		_, err = srv.CreateOrganizationUser(ctx, &v1.CreateOrganizationUserRequest{
			OrganizationId: cresp.Id,
			UserId:         "user 1",
			Role:           v1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
		})
		assert.NoError(t, err)
	}

	lresp, err := srv.ListOrganizations(ctx, &v1.ListOrganizationsRequest{})
	assert.NoError(t, err)
	assert.Len(t, lresp.Organizations, 2)

	ilresp, err := isrv.ListInternalOrganizations(ctx, &v1.ListInternalOrganizationsRequest{})
	assert.NoError(t, err)
	assert.Len(t, ilresp.Organizations, 2)

	laresp, err := isrv.store.ListAllOrganizationUsers()
	assert.NoError(t, err)
	assert.Len(t, laresp, 2)

	_, err = srv.DeleteOrganization(ctx, &v1.DeleteOrganizationRequest{
		Id: lresp.Organizations[0].Id,
	})
	assert.NoError(t, err)

	lresp2, err := srv.ListOrganizations(ctx, &v1.ListOrganizationsRequest{})
	assert.NoError(t, err)
	assert.Len(t, lresp2.Organizations, 1)
	assert.Equal(t, lresp2.Organizations[0].Id, lresp.Organizations[1].Id)

	laresp2, err := isrv.ListOrganizationUsers(ctx, &v1.ListOrganizationUsersRequest{})
	assert.NoError(t, err)
	assert.Len(t, laresp2.Users, 1)

	_, err = srv.DeleteOrganizationUser(ctx, &v1.DeleteOrganizationUserRequest{
		OrganizationId: lresp2.Organizations[0].Id,
		UserId:         "user 1",
	})
	assert.NoError(t, err)

	laresp3, err := isrv.ListOrganizationUsers(ctx, &v1.ListOrganizationUsersRequest{})
	assert.NoError(t, err)
	assert.Empty(t, laresp3.Users)
}

func TestDeleteOrganization(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	ctx := context.Background()

	org, err := srv.CreateOrganization(ctx, &v1.CreateOrganizationRequest{
		Title: "Test organization",
	})
	assert.NoError(t, err)

	proj, err := srv.CreateProject(ctx, &v1.CreateProjectRequest{
		Title:               "Test project",
		OrganizationId:      org.Id,
		KubernetesNamespace: "test",
	})
	assert.NoError(t, err)

	_, err = srv.DeleteOrganization(ctx, &v1.DeleteOrganizationRequest{
		Id: org.Id,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	// Delete the project and try again.
	_, err = srv.DeleteProject(ctx, &v1.DeleteProjectRequest{
		Id:             proj.Id,
		OrganizationId: org.Id,
	})
	assert.NoError(t, err)

	_, err = srv.DeleteOrganization(ctx, &v1.DeleteOrganizationRequest{
		Id: org.Id,
	})
	assert.NoError(t, err)
}

func TestCreateOrganization_UniqueConstraintViolation(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	ctx := context.Background()

	_, err := srv.CreateOrganization(ctx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.NoError(t, err)

	_, err = srv.CreateOrganization(ctx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestCreateOrganizationUser_UniqueConstraintViolation(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	ctx := context.Background()

	o, err := srv.CreateOrganization(ctx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.NoError(t, err)

	_, err = srv.CreateOrganizationUser(ctx, &v1.CreateOrganizationUserRequest{
		OrganizationId: o.Id,
		UserId:         "u0",
		Role:           v1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
	})
	assert.NoError(t, err)

	_, err = srv.CreateOrganizationUser(ctx, &v1.CreateOrganizationUserRequest{
		OrganizationId: o.Id,
		UserId:         "u0",
		Role:           v1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestListOrganizationUsers(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	ctx := context.Background()

	var orgs []*v1.Organization
	for i := 0; i < 2; i++ {
		org, err := srv.CreateOrganization(ctx, &v1.CreateOrganizationRequest{
			Title: fmt.Sprintf("title %d", i),
		})
		assert.NoError(t, err)
		orgs = append(orgs, org)

		// Delete the default user to make the rest of the test simple.
		_, err = srv.DeleteOrganizationUser(ctx, &v1.DeleteOrganizationUserRequest{
			OrganizationId: org.Id,
			UserId:         defaultUserID,
		})
		assert.NoError(t, err)

		_, err = srv.CreateOrganizationUser(ctx, &v1.CreateOrganizationUserRequest{
			OrganizationId: org.Id,
			UserId:         fmt.Sprintf("user %d", i),
			Role:           v1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
		})
		assert.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		resp, err := srv.ListOrganizationUsers(ctx, &v1.ListOrganizationUsersRequest{OrganizationId: orgs[i].Id})
		assert.NoError(t, err)
		assert.Len(t, resp.Users, 1)
		assert.Equal(t, fmt.Sprintf("user %d", i), resp.Users[0].UserId)
	}
}

func TestDeleteDeleteOrganizationUser(t *testing.T) {
	const userID = "u0"

	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	ctx := context.Background()

	o, err := srv.CreateOrganization(ctx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.NoError(t, err)

	// Delete the default user to make the rest of the test simple.
	_, err = srv.DeleteOrganizationUser(ctx, &v1.DeleteOrganizationUserRequest{
		OrganizationId: o.Id,
		UserId:         defaultUserID,
	})
	assert.NoError(t, err)

	p, err := srv.CreateProject(ctx, &v1.CreateProjectRequest{
		Title:               "title",
		OrganizationId:      o.Id,
		KubernetesNamespace: "ns",
	})
	assert.NoError(t, err)

	_, err = srv.CreateOrganizationUser(ctx, &v1.CreateOrganizationUserRequest{
		OrganizationId: o.Id,
		UserId:         userID,
		Role:           v1.OrganizationRole_ORGANIZATION_ROLE_OWNER,
	})
	assert.NoError(t, err)

	_, err = srv.CreateProjectUser(ctx, &v1.CreateProjectUserRequest{
		ProjectId:      p.Id,
		OrganizationId: o.Id,
		UserId:         userID,
		Role:           v1.ProjectRole_PROJECT_ROLE_OWNER,
	})
	assert.NoError(t, err)

	resp, err := srv.ListProjectUsers(ctx, &v1.ListProjectUsersRequest{
		ProjectId:      p.Id,
		OrganizationId: o.Id,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Users, 1)
	assert.Equal(t, resp.Users[0].UserId, userID)

	// Delete the org user. Make sure the project user is deleted as well.
	_, err = srv.DeleteOrganizationUser(ctx, &v1.DeleteOrganizationUserRequest{
		OrganizationId: o.Id,
		UserId:         userID,
	})
	assert.NoError(t, err)

	resp, err = srv.ListProjectUsers(ctx, &v1.ListProjectUsersRequest{
		ProjectId:      p.Id,
		OrganizationId: o.Id,
	})
	assert.NoError(t, err)
	assert.Empty(t, resp.Users)
}

func TestCreateDefaultOrganization(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	c := &config.DefaultOrganizationConfig{
		Title: "default",
		UserIDs: []string{
			"admin",
		},
		TenantID: defaultTenantID,
	}
	created, err := srv.CreateDefaultOrganization(context.Background(), c)
	assert.NoError(t, err)
	assert.Equal(t, created.Title, c.Title)
	assert.True(t, created.IsDefault)

	o, err := st.GetDefaultOrganization(c.TenantID)
	assert.NoError(t, err)

	users, err := st.ListAllOrganizationUsers()
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	u := users[0]
	assert.Equal(t, o.OrganizationID, u.OrganizationID)
	assert.Equal(t, "admin", u.UserID)
	assert.Equal(t, v1.OrganizationRole_ORGANIZATION_ROLE_OWNER.String(), u.Role)

	// Calling again is no-op.
	_, err = srv.CreateDefaultOrganization(context.Background(), c)
	assert.NoError(t, err)

	// Default org cannot be deleted.
	_, err = srv.DeleteOrganization(context.Background(), &v1.DeleteOrganizationRequest{
		Id: o.OrganizationID,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateOrganization_EnableAuth(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	srv.enableAuth = true

	createDefaultOrg(t, srv, "admin")

	ui := auth.UserInfo{
		UserID: "non-admin",
	}
	adminCtx := auth.AppendUserInfoToContext(context.Background(), ui)
	_, err := srv.CreateOrganization(adminCtx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	ui = auth.UserInfo{
		UserID: "admin",
	}
	nonadminCtx := auth.AppendUserInfoToContext(context.Background(), ui)
	_, err = srv.CreateOrganization(nonadminCtx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.NoError(t, err)
}

func TestListOrganization_EnableAuth(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	srv.enableAuth = true

	o0 := createDefaultOrg(t, srv, "user0")

	ui := auth.UserInfo{
		UserID: "user0",
	}
	u0Ctx := auth.AppendUserInfoToContext(context.Background(), ui)

	o1, err := srv.CreateOrganization(u0Ctx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.NoError(t, err)

	_, err = srv.CreateOrganizationUser(u0Ctx, &v1.CreateOrganizationUserRequest{
		OrganizationId: o1.Id,
		UserId:         "user1",
		Role:           v1.OrganizationRole_ORGANIZATION_ROLE_READER,
	})
	assert.NoError(t, err)

	resp, err := srv.ListOrganizations(u0Ctx, &v1.ListOrganizationsRequest{})
	assert.NoError(t, err)
	assert.Len(t, resp.Organizations, 2)
	var ids []string
	for _, o := range resp.Organizations {
		ids = append(ids, o.Id)
	}
	assert.ElementsMatch(t, []string{o0.OrganizationID, o1.Id}, ids)

	ui = auth.UserInfo{
		UserID: "user1",
	}
	u1Ctx := auth.AppendUserInfoToContext(context.Background(), ui)
	resp, err = srv.ListOrganizations(u1Ctx, &v1.ListOrganizationsRequest{})
	assert.NoError(t, err)
	assert.Len(t, resp.Organizations, 1)
	assert.Equal(t, resp.Organizations[0].Id, o1.Id)

	ui = auth.UserInfo{
		UserID: "u2",
	}
	u2Ctx := auth.AppendUserInfoToContext(context.Background(), ui)
	resp, err = srv.ListOrganizations(u2Ctx, &v1.ListOrganizationsRequest{})
	assert.NoError(t, err)
	assert.Empty(t, resp.Organizations)
}

func TestDeleteOrganization_EnableAuth(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	srv.enableAuth = true

	createDefaultOrg(t, srv, "user0")

	ui := auth.UserInfo{
		UserID: "user0",
	}
	u0Ctx := auth.AppendUserInfoToContext(context.Background(), ui)

	o1, err := srv.CreateOrganization(u0Ctx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.NoError(t, err)

	ui = auth.UserInfo{
		UserID: "user1",
	}
	u1Ctx := auth.AppendUserInfoToContext(context.Background(), ui)

	_, err = srv.DeleteOrganization(u1Ctx, &v1.DeleteOrganizationRequest{
		Id: o1.Id,
	})
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	_, err = srv.DeleteOrganization(u0Ctx, &v1.DeleteOrganizationRequest{
		Id: o1.Id,
	})
	assert.NoError(t, err)
}

func TestOrganizationUser_EnableAuth(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st)
	srv.enableAuth = true

	createDefaultOrg(t, srv, "user0")

	ui := auth.UserInfo{
		UserID: "user0",
	}
	u0Ctx := auth.AppendUserInfoToContext(context.Background(), ui)

	o1, err := srv.CreateOrganization(u0Ctx, &v1.CreateOrganizationRequest{
		Title: "title",
	})
	assert.NoError(t, err)

	ui = auth.UserInfo{
		UserID: "user1",
	}
	u1Ctx := auth.AppendUserInfoToContext(context.Background(), ui)

	creq := &v1.CreateOrganizationUserRequest{
		OrganizationId: o1.Id,
		UserId:         "some user",
		Role:           v1.OrganizationRole_ORGANIZATION_ROLE_READER,
	}
	_, err = srv.CreateOrganizationUser(u1Ctx, creq)
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	_, err = srv.CreateOrganizationUser(u0Ctx, creq)
	assert.NoError(t, err)

	lreq := &v1.ListOrganizationUsersRequest{
		OrganizationId: o1.Id,
	}
	_, err = srv.ListOrganizationUsers(u1Ctx, lreq)
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	_, err = srv.ListOrganizationUsers(u0Ctx, lreq)
	assert.NoError(t, err)

	dreq := &v1.DeleteOrganizationUserRequest{
		OrganizationId: o1.Id,
		UserId:         "some user",
	}
	_, err = srv.DeleteOrganizationUser(u1Ctx, dreq)
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	_, err = srv.DeleteOrganizationUser(u0Ctx, dreq)
	assert.NoError(t, err)
}

func createDefaultOrg(t *testing.T, srv *S, userID string) *store.Organization {
	c := &config.DefaultOrganizationConfig{
		Title:   "default",
		UserIDs: []string{userID},
	}
	o, err := srv.CreateDefaultOrganization(context.Background(), c)
	assert.NoError(t, err)
	return o
}
