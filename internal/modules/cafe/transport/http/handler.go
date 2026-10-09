package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/cafe/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/cafe/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Handler struct {
	s *application.Service
	auth usersapi.Authenticator
	workspace organizationapi.WorkspaceResolver
	access organizationapi.WorkspaceAccess
}
func New(s *application.Service,auth usersapi.Authenticator,workspace organizationapi.WorkspaceResolver,access organizationapi.WorkspaceAccess)*Handler{
	return &Handler{s:s,auth:auth,workspace:workspace,access:access}
}
func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	current:=routes.With(usersapi.RequireAuthentication(h.auth),organizationapi.RequireWorkspace(h.workspace))
	current.With(organizationapi.RequireWorkspacePermission(h.access,"cafe.booth.read")).Get("/cafe/booths",h.booths)
	current.With(organizationapi.RequireWorkspacePermission(h.access,"cafe.booth.manage")).Post("/cafe/booths",h.createBooth)
	current.With(organizationapi.RequireWorkspacePermission(h.access,"cafe.booking.read")).Get("/cafe/bookings",h.bookings)
	current.With(organizationapi.RequireWorkspacePermission(h.access,"cafe.booking.create")).Post("/cafe/bookings",h.reserve)
	bookingManage:=current.With(organizationapi.RequireWorkspacePermission(h.access,"cafe.booking.manage"))
	bookingManage.Post("/cafe/bookings/{id}/check-in",h.changeBooking("check-in"))
	bookingManage.Post("/cafe/bookings/{id}/complete",h.changeBooking("complete"))
	bookingManage.Post("/cafe/bookings/{id}/cancel",h.changeBooking("cancel"))
	current.With(organizationapi.RequireWorkspacePermission(h.access,"cafe.order.read")).Get("/cafe/orders",h.orders)
	current.With(organizationapi.RequireWorkspacePermission(h.access,"cafe.order.create")).Post("/cafe/orders",h.createOrder)
	orderManage:=current.With(organizationapi.RequireWorkspacePermission(h.access,"cafe.order.manage"))
	orderManage.Post("/cafe/orders/{id}/complete",h.changeOrder("complete"))
	orderManage.Post("/cafe/orders/{id}/cancel",h.changeOrder("cancel"))
	return nil
}
func ids(r *http.Request)(uuid.UUID,uuid.UUID){
	w,_:=organizationapi.WorkspaceContextFromContext(r.Context())
	p,_:=usersapi.PrincipalFromContext(r.Context())
	return w.WorkspaceID,p.UserID
}
func decode(w http.ResponseWriter,r *http.Request,v any)bool{
	dec:=json.NewDecoder(http.MaxBytesReader(w,r.Body,1<<20))
	dec.DisallowUnknownFields()
	if err:=dec.Decode(v);err!=nil{httpserver.WriteError(w,r,400,"VALIDATION_ERROR","Invalid request body");return false}
	var trailing any
	if err:=dec.Decode(&trailing);err!=io.EOF{httpserver.WriteError(w,r,400,"VALIDATION_ERROR","Expected one JSON object");return false}
	return true
}
func send(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func fail(w http.ResponseWriter,r *http.Request,err error){
	switch {
	case errors.Is(err,domain.ErrInvalid): httpserver.WriteError(w,r,400,"VALIDATION_ERROR","Invalid café input or inactive catalog item")
	case errors.Is(err,domain.ErrNotFound): httpserver.WriteError(w,r,404,"NOT_FOUND","Record not found for this workspace")
	case errors.Is(err,domain.ErrConflict): httpserver.WriteError(w,r,409,"CONFLICT","Unavailable slot, duplicated booth, or invalid status transition")
	default:httpserver.WriteApplicationError(w,r,err)
	}
}
func pathID(w http.ResponseWriter,r *http.Request)(uuid.UUID,bool){
	id,err:=uuid.Parse(chi.URLParam(r,"id"))
	if err!=nil{httpserver.WriteError(w,r,400,"VALIDATION_ERROR","Invalid record identifier");return uuid.Nil,false}
	return id,true
}
func (h *Handler) booths(w http.ResponseWriter,r *http.Request) {
	ws,_:=ids(r)
	items,err:=h.s.ListBooths(r.Context(),ws)
	if err!=nil{fail(w,r,err);return}
	send(w,200,items)
}
func (h *Handler) createBooth(w http.ResponseWriter,r *http.Request) {
	var body struct{Name string `json:"name"`}
	if !decode(w,r,&body){return}
	ws,actor:=ids(r)
	item,err:=h.s.CreateBooth(r.Context(),ws,actor,body.Name)
	if err!=nil{fail(w,r,err);return}
	send(w,201,item)
}
func (h *Handler) bookings(w http.ResponseWriter,r *http.Request) {
	ws,_:=ids(r)
	items,err:=h.s.ListBookings(r.Context(),ws)
	if err!=nil{fail(w,r,err);return}
	send(w,200,items)
}
func (h *Handler) reserve(w http.ResponseWriter,r *http.Request) {
	var body domain.BookingInput
	if !decode(w,r,&body){return}
	ws,actor:=ids(r)
	item,err:=h.s.Reserve(r.Context(),ws,actor,body)
	if err!=nil{fail(w,r,err);return}
	send(w,201,item)
}
func (h *Handler) changeBooking(action string) http.HandlerFunc{
	return func(w http.ResponseWriter,r *http.Request){
		id,ok:=pathID(w,r);if !ok{return}
		ws,_:=ids(r)
		item,err:=h.s.TransitionBooking(r.Context(),ws,id,action)
		if err!=nil{fail(w,r,err);return}
		send(w,200,item)
	}
}
func (h *Handler) orders(w http.ResponseWriter,r *http.Request) {
	ws,_:=ids(r)
	items,err:=h.s.ListOrders(r.Context(),ws)
	if err!=nil{fail(w,r,err);return}
	send(w,200,items)
}
func (h *Handler) createOrder(w http.ResponseWriter,r *http.Request) {
	var body domain.OrderInput
	if !decode(w,r,&body){return}
	ws,actor:=ids(r)
	item,err:=h.s.CreateOrder(r.Context(),ws,actor,body)
	if err!=nil{fail(w,r,err);return}
	send(w,201,item)
}
func (h *Handler) changeOrder(action string) http.HandlerFunc{
	return func(w http.ResponseWriter,r *http.Request){
		id,ok:=pathID(w,r);if !ok{return}
		ws,_:=ids(r)
		item,err:=h.s.TransitionOrder(r.Context(),ws,id,action)
		if err!=nil{fail(w,r,err);return}
		send(w,200,item)
	}
}
