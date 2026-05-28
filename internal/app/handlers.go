package app

import (
	"io/fs"
	"net/http"

	httpmiddleware "github.com/lqhiyul/personality-type-test/internal/http/middleware"
	"github.com/lqhiyul/personality-type-test/internal/sessions"
)

type App struct {
	store            *Store
	userStore        *UserStore
	adminPassword    string
	cookieSecure     bool
	sessionName      string
	baseTemplateFS   fs.FS
	loginLimiter     *loginRateLimiter
	userLoginLimiter *loginRateLimiter
	sessionStore     *sessions.Store
	auditStore       *AdminAuditStore
	trustedProxies   trustedProxySet
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	a.registerPublicRoutes(mux)
	a.registerAuthRoutes(mux)
	a.registerCurrentUserRoutes(mux)
	a.registerSafetyRoutes(mux)
	a.registerSocialRoutes(mux)
	a.registerAdminRoutes(mux)
	a.registerStaticRoutes(mux)

	return httpmiddleware.RequestID(httpmiddleware.SecurityHeaders(httpmiddleware.CSRF{SecureCookie: a.cookieSecure}.Middleware(mux)))
}

func (a *App) registerPublicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", a.handleHealthz)
	mux.HandleFunc("/api/submit", a.handleSubmit)
	mux.HandleFunc("/api/users/", a.handlePublicUserProfile)
}

func (a *App) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/register", a.handleAuthRegister)
	mux.HandleFunc("/api/auth/login", a.handleAuthLogin)
	mux.HandleFunc("/api/auth/logout", a.handleAuthLogout)
	mux.HandleFunc("/api/auth/me", a.handleAuthMe)
}

func (a *App) registerCurrentUserRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/me/profile", a.handleMyProfile)
	mux.HandleFunc("/api/me/results/", a.handleMyResultByID)
	mux.HandleFunc("/api/me/results", a.handleMyResults)
}

func (a *App) registerSafetyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/blocks/", a.handleBlockByUsername)
	mux.HandleFunc("/api/blocks", a.handleBlocks)
	mux.HandleFunc("/api/reports", a.handleCreateReport)
	mux.HandleFunc("/api/admin/reports/", a.handleAdminReportByID)
	mux.HandleFunc("/api/admin/reports", a.handleAdminReports)
}

func (a *App) registerSocialRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/messages/start", a.handleStartMessageConversation)
	mux.HandleFunc("/api/messages/conversations/", a.handleMessageConversationByID)
	mux.HandleFunc("/api/messages/conversations", a.handleMessageConversations)
	mux.HandleFunc("/api/messages/", a.handleMessageByID)
	mux.HandleFunc("/api/profile-comments/", a.handleProfileCommentByID)
	mux.HandleFunc("/api/friends/requests/", a.handleFriendRequestByID)
	mux.HandleFunc("/api/friends/requests", a.handleFriendRequests)
	mux.HandleFunc("/api/friends/request", a.handleCreateFriendRequest)
	mux.HandleFunc("/api/friends/", a.handleFriendByID)
	mux.HandleFunc("/api/friends", a.handleFriends)
}

func (a *App) registerAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/login", a.handleLogin)
	mux.HandleFunc("/api/logout", a.handleLogout)
	mux.HandleFunc("/api/results/export", a.handleExportResults)
	mux.HandleFunc("/api/results/", a.handleResultByID)
	mux.HandleFunc("/api/results", a.handleResults)
	mux.HandleFunc("/api/stats", a.handleStats)
}

func (a *App) registerStaticRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", a.handleStatic)
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.routes().ServeHTTP(w, r)
}
