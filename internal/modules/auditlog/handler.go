package auditlog

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	"rt26/backend/internal/pkg/helper"
	"rt26/backend/internal/pkg/response"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

type scanRow struct {
	ID          string  `gorm:"column:id"`
	UserID      *string `gorm:"column:user_id"`
	NamaUser    string  `gorm:"column:nama_user"`
	RoleUser    string  `gorm:"column:role_user"`
	Modul       string  `gorm:"column:modul"`
	Aktivitas   string  `gorm:"column:aktivitas"`
	Deskripsi   string  `gorm:"column:deskripsi"`
	DataSebelum []byte  `gorm:"column:data_sebelum"`
	DataSesudah []byte  `gorm:"column:data_sesudah"`
	IPAddress   string    `gorm:"column:ip_address"`
	UserAgent   string    `gorm:"column:user_agent"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (s *scanRow) toResponse() AuditLogResponse {
	r := AuditLogResponse{
		ID:        s.ID,
		UserID:    s.UserID,
		NamaUser:  s.NamaUser,
		RoleUser:  s.RoleUser,
		Modul:     s.Modul,
		Aktivitas: s.Aktivitas,
		Deskripsi: s.Deskripsi,
		IPAddress: s.IPAddress,
		UserAgent: s.UserAgent,
		CreatedAt: s.CreatedAt,
	}
	if len(s.DataSebelum) > 0 {
		json.Unmarshal(s.DataSebelum, &r.DataSebelum)
	}
	if len(s.DataSesudah) > 0 {
		json.Unmarshal(s.DataSesudah, &r.DataSesudah)
	}
	return r
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pg := helper.ParsePagination(r, 10)
	search := helper.CleanSearch(r.URL.Query().Get("search"))
	modul := r.URL.Query().Get("modul")
	aktivitas := r.URL.Query().Get("aktivitas")
	userID := r.URL.Query().Get("user_id")
	tanggalMulai := r.URL.Query().Get("tanggal_mulai")
	tanggalAkhir := r.URL.Query().Get("tanggal_akhir")

	query := h.DB.WithContext(r.Context()).
		Table("audit_logs").
		Select("audit_logs.*, COALESCE(u.nama, '') as nama_user, COALESCE(r.nama, '') as role_user").
		Joins("LEFT JOIN users u ON u.id = audit_logs.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL")

	countQuery := h.DB.WithContext(r.Context()).
		Table("audit_logs").
		Joins("LEFT JOIN users u ON u.id = audit_logs.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		condition := "(u.nama ILIKE ? OR audit_logs.modul ILIKE ? OR audit_logs.aktivitas ILIKE ? OR audit_logs.deskripsi ILIKE ?)"
		query = query.Where(condition, like, like, like, like)
		countQuery = countQuery.Where(condition, like, like, like, like)
	}
	if modul != "" {
		query = query.Where("audit_logs.modul ILIKE ?", modul)
		countQuery = countQuery.Where("audit_logs.modul ILIKE ?", modul)
	}
	if aktivitas != "" {
		query = query.Where("audit_logs.aktivitas ILIKE ?", aktivitas)
		countQuery = countQuery.Where("audit_logs.aktivitas ILIKE ?", aktivitas)
	}
	if userID != "" {
		query = query.Where("audit_logs.user_id = ?", userID)
		countQuery = countQuery.Where("audit_logs.user_id = ?", userID)
	}
	if tanggalMulai != "" {
		query = query.Where("audit_logs.created_at >= ?", tanggalMulai)
		countQuery = countQuery.Where("audit_logs.created_at >= ?", tanggalMulai)
	}
	if tanggalAkhir != "" {
		query = query.Where("audit_logs.created_at <= ?", tanggalAkhir+" 23:59:59")
		countQuery = countQuery.Where("audit_logs.created_at <= ?", tanggalAkhir+" 23:59:59")
	}

	var total int64
	countQuery.Count(&total)

	switch pg.Sort {
	case "created_at":
		query = query.Order("audit_logs.created_at ASC")
	case "-created_at":
		query = query.Order("audit_logs.created_at DESC")
	default:
		query = query.Order("audit_logs.created_at DESC")
	}

	var rows []scanRow
	if err := query.Offset(pg.Offset()).Limit(pg.Limit).Scan(&rows).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat audit log")
		return
	}

	items := make([]AuditLogResponse, len(rows))
	for i, row := range rows {
		items[i] = row.toResponse()
	}

	response.SuccessPaginated(w, "Audit log berhasil dimuat", items, response.Pagination{
		Page:      pg.Page,
		Limit:     pg.Limit,
		TotalData: total,
		TotalPage: pg.TotalPage(total),
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "ID wajib diisi")
		return
	}

	var row scanRow
	if err := h.DB.WithContext(r.Context()).
		Table("audit_logs").
		Select("audit_logs.*, COALESCE(u.nama, '') as nama_user, COALESCE(r.nama, '') as role_user").
		Joins("LEFT JOIN users u ON u.id = audit_logs.user_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN roles r ON r.id = u.role_id AND r.deleted_at IS NULL").
		Where("audit_logs.id = ?", id).
		Scan(&row).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat audit log")
		return
	}

	if row.ID == "" {
		response.Error(w, http.StatusNotFound, "Audit log tidak ditemukan")
		return
	}

	resp := row.toResponse()
	response.Success(w, http.StatusOK, "Audit log berhasil dimuat", resp)
}
