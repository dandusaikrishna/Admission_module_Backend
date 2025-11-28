package handlers

import (
"admission-module/db"
"admission-module/http/response"
"admission-module/models"
"encoding/json"
"fmt"
"log"
"net/http"
"strconv"
"time"
)

// GetCounsellors retrieves all counsellors or a specific one by ID with their stats and leads
func GetCounsellors(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodGet {
response.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
return
}

// Check if ID is provided in query parameter
counsellorIDStr := r.URL.Query().Get("id")

if counsellorIDStr != "" {
// Get specific counsellor by ID
getCounsellorByID(w, r, counsellorIDStr)
} else {
// Get all counsellors
getAllCounsellors(w, r)
}
}

// getAllCounsellors retrieves all counsellors with their stats
func getAllCounsellors(w http.ResponseWriter, r *http.Request) {
query := `
SELECT 
c.id, 
c.name, 
c.email, 
c.assigned_count, 
c.max_capacity,
COUNT(sl.id) as active_leads
FROM counselor c
LEFT JOIN student_lead sl ON c.id = sl.counselor_id
GROUP BY c.id, c.name, c.email, c.assigned_count, c.max_capacity
ORDER BY c.name ASC
`

rows, err := db.DB.QueryContext(r.Context(), query)
if err != nil {
log.Printf("Error fetching counsellors: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error fetching counsellours")
return
}
defer rows.Close()

counsellors := []map[string]interface{}{}
for rows.Next() {
var id, assignedCount, maxCapacity, activeLeads int
var name, email string
if err := rows.Scan(&id, &name, &email, &assignedCount, &maxCapacity, &activeLeads); err != nil {
log.Printf("Error scanning counsellor: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error processing counsellours")
return
}

counsellors = append(counsellors, map[string]interface{}{
"id":              id,
"name":            name,
"email":           email,
"assigned_count":  assignedCount,
"max_capacity":    maxCapacity,
"active_leads":    activeLeads,
"available_slots": maxCapacity - assignedCount,
"utilization":     float64(assignedCount) / float64(maxCapacity) * 100,
})
}

if err = rows.Err(); err != nil {
log.Printf("Error iterating counsellours: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error processing counsellours")
return
}

response.SuccessResponse(w, http.StatusOK, fmt.Sprintf("Retrieved %d counsellours", len(counsellors)), counsellors)
}

// getCounsellorByID retrieves a specific counsellor with their leads
func getCounsellorByID(w http.ResponseWriter, r *http.Request, counsellorIDStr string) {
counsellorID, err := strconv.Atoi(counsellorIDStr)
if err != nil {
response.ErrorResponse(w, http.StatusBadRequest, "Invalid counsellor ID")
return
}

// Get counsellor details
var counsellor models.Counsellor
counsellorQuery := `
SELECT id, name, email, assigned_count, max_capacity
FROM counselor
WHERE id = $1
`
err = db.DB.QueryRowContext(r.Context(), counsellorQuery, counsellorID).
Scan(&counsellor.ID, &counsellor.Name, &counsellor.Email, &counsellor.AssignedCount, &counsellor.MaxCapacity)
if err != nil {
log.Printf("Error fetching counsellour: %v", err)
response.ErrorResponse(w, http.StatusNotFound, "Counsellor not found")
return
}

// Get all leads assigned to this counsellor
leadsQuery := `
SELECT 
id, name, email, phone, education, lead_source, 
counselor_id, registration_fee_status, course_fee_status,
application_status, interview_scheduled_at, created_at
FROM student_lead
WHERE counselor_id = $1
ORDER BY created_at DESC
`
leadsRows, err := db.DB.QueryContext(r.Context(), leadsQuery, counsellorID)
if err != nil {
log.Printf("Error fetching leads: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error fetching leads")
return
}
defer leadsRows.Close()

leads := []map[string]interface{}{}
for leadsRows.Next() {
var id, counselorID int
var name, email, phone, education, leadSource string
var registrationFeeStatus, courseFeeStatus, applicationStatus string
var interviewScheduledAt *time.Time
var createdAt time.Time

if err := leadsRows.Scan(
&id, &name, &email, &phone, &education, &leadSource,
&counselorID, &registrationFeeStatus, &courseFeeStatus,
&applicationStatus, &interviewScheduledAt, &createdAt,
); err != nil {
log.Printf("Error scanning lead: %v", err)
continue
}

leads = append(leads, map[string]interface{}{
"id":                      id,
"name":                    name,
"email":                   email,
"phone":                   phone,
"education":               education,
"lead_source":             leadSource,
"counselor_id":            counselorID,
"registration_fee_status": registrationFeeStatus,
"course_fee_status":       courseFeeStatus,
"application_status":      applicationStatus,
"interview_scheduled_at":  interviewScheduledAt,
"created_at":              createdAt,
})
}

if err = leadsRows.Err(); err != nil {
log.Printf("Error iterating leads: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error processing leads")
return
}

result := map[string]interface{}{
"counsellor": map[string]interface{}{
"id":              counsellor.ID,
"name":            counsellor.Name,
"email":           counsellor.Email,
"assigned_count":  counsellor.AssignedCount,
"max_capacity":    counsellor.MaxCapacity,
"available_slots": counsellor.MaxCapacity - counsellor.AssignedCount,
"utilization":     float64(counsellor.AssignedCount) / float64(counsellor.MaxCapacity) * 100,
},
"leads": leads,
}

response.SuccessResponse(w, http.StatusOK, "Counsellor details retrieved", result)
}

// CreateCounsellor creates a new counsellor
func CreateCounsellor(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodPost {
response.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
return
}

var req struct {
Name        string `json:"name"`
Email       string `json:"email"`
Phone       string `json:"phone"`
MaxCapacity int    `json:"max_capacity"`
}

if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
response.ErrorResponse(w, http.StatusBadRequest, "Invalid request: "+err.Error())
return
}

if req.Name == "" || req.Email == "" {
response.ErrorResponse(w, http.StatusBadRequest, "Name and email are required")
return
}

if req.MaxCapacity <= 0 {
req.MaxCapacity = 10
}

now := time.Now()
var counsellorID int
query := `
INSERT INTO counselor (name, email, phone, max_capacity, assigned_count, created_at, updated_at)
VALUES ($1, $2, $3, $4, 0, $5, $6)
RETURNING id
`

err := db.DB.QueryRowContext(r.Context(), query, req.Name, req.Email, req.Phone, req.MaxCapacity, now, now).Scan(&counsellorID)
if err != nil {
log.Printf("Error creating counsellor: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error creating counsellor")
return
}

response.SuccessResponse(w, http.StatusCreated, "Counsellor created successfully", map[string]interface{}{
"id":           counsellorID,
"name":         req.Name,
"email":        req.Email,
"max_capacity": req.MaxCapacity,
})
}

// UpdateCounsellor updates an existing counsellor
func UpdateCounsellor(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodPut {
response.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
return
}

var req struct {
ID          int    `json:"id"`
Name        string `json:"name"`
Email       string `json:"email"`
Phone       string `json:"phone"`
MaxCapacity int    `json:"max_capacity"`
}

if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
response.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
return
}

if req.ID == 0 {
response.ErrorResponse(w, http.StatusBadRequest, "Counsellor ID is required")
return
}

query := `
UPDATE counselor
SET name = $1, email = $2, phone = $3, max_capacity = $4, updated_at = $5
WHERE id = $6
`

result, err := db.DB.ExecContext(r.Context(), query, req.Name, req.Email, req.Phone, req.MaxCapacity, time.Now(), req.ID)
if err != nil {
log.Printf("Error updating counsellor: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error updating counsellor")
return
}

rowsAffected, err := result.RowsAffected()
if err != nil {
response.ErrorResponse(w, http.StatusInternalServerError, "Error checking update")
return
}

if rowsAffected == 0 {
response.ErrorResponse(w, http.StatusNotFound, "Counsellor not found")
return
}

response.SuccessResponse(w, http.StatusOK, "Counsellor updated successfully", map[string]interface{}{
"counsellor_id": req.ID,
})
}

// DeleteCounsellor deletes a counsellor (only if no leads assigned)
func DeleteCounsellor(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodDelete {
response.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
return
}

counsellorIDStr := r.URL.Query().Get("id")
if counsellorIDStr == "" {
response.ErrorResponse(w, http.StatusBadRequest, "Counsellor ID is required")
return
}

counsellorID, err := strconv.Atoi(counsellorIDStr)
if err != nil {
response.ErrorResponse(w, http.StatusBadRequest, "Invalid counsellor ID")
return
}

var leadCount int
checkQuery := `SELECT COUNT(*) FROM student_lead WHERE counselor_id = $1`
err = db.DB.QueryRowContext(r.Context(), checkQuery, counsellorID).Scan(&leadCount)
if err != nil {
log.Printf("Error checking leads: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error checking counsellor leads")
return
}

if leadCount > 0 {
response.ErrorResponse(w, http.StatusConflict, fmt.Sprintf("Cannot delete counsellor with %d assigned leads", leadCount))
return
}

query := `DELETE FROM counselor WHERE id = $1`
result, err := db.DB.ExecContext(r.Context(), query, counsellorID)
if err != nil {
log.Printf("Error deleting counsellor: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error deleting counsellor")
return
}

rowsAffected, err := result.RowsAffected()
if err != nil {
response.ErrorResponse(w, http.StatusInternalServerError, "Error checking delete")
return
}

if rowsAffected == 0 {
response.ErrorResponse(w, http.StatusNotFound, "Counsellor not found")
return
}

response.SuccessResponse(w, http.StatusOK, "Counsellor deleted successfully", map[string]interface{}{
"counsellor_id": counsellorID,
})
}

// AssignLeadToCounsellor assigns a lead to a counsellor
func AssignLeadToCounsellor(w http.ResponseWriter, r *http.Request) {
if r.Method != http.MethodPost {
response.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
return
}

var req struct {
LeadID       int `json:"lead_id"`
CounsellorID int `json:"counsellor_id"`
}

if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
response.ErrorResponse(w, http.StatusBadRequest, "Invalid request: "+err.Error())
return
}

if req.LeadID == 0 || req.CounsellorID == 0 {
response.ErrorResponse(w, http.StatusBadRequest, "Lead ID and Counsellor ID are required")
return
}

var currentCapacity, maxCapacity int
capacityQuery := `SELECT assigned_count, max_capacity FROM counselor WHERE id = $1`
err := db.DB.QueryRowContext(r.Context(), capacityQuery, req.CounsellorID).Scan(&currentCapacity, &maxCapacity)
if err != nil {
log.Printf("Error fetching counsellor capacity: %v", err)
response.ErrorResponse(w, http.StatusNotFound, "Counsellor not found")
return
}

if currentCapacity >= maxCapacity {
response.ErrorResponse(w, http.StatusConflict, "Counsellor has reached maximum capacity")
return
}

tx, err := db.DB.BeginTx(r.Context(), nil)
if err != nil {
log.Printf("Error starting transaction: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error starting transaction")
return
}
defer tx.Rollback()

updateLeadQuery := `UPDATE student_lead SET counselor_id = $1, updated_at = $2 WHERE id = $3`
_, err = tx.ExecContext(r.Context(), updateLeadQuery, req.CounsellorID, time.Now(), req.LeadID)
if err != nil {
log.Printf("Error updating lead: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error updating lead")
return
}

updateCounsellorQuery := `UPDATE counselor SET assigned_count = assigned_count + 1, updated_at = $1 WHERE id = $2`
_, err = tx.ExecContext(r.Context(), updateCounsellorQuery, time.Now(), req.CounsellorID)
if err != nil {
log.Printf("Error updating counsellor: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error updating counsellor")
return
}

err = tx.Commit()
if err != nil {
log.Printf("Error committing transaction: %v", err)
response.ErrorResponse(w, http.StatusInternalServerError, "Error committing transaction")
return
}

response.SuccessResponse(w, http.StatusOK, "Lead assigned successfully", map[string]interface{}{
"lead_id":       req.LeadID,
"counsellor_id": req.CounsellorID,
})
}