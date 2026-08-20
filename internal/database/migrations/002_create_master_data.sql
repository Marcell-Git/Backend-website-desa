CREATE TABLE IF NOT EXISTS anggota_keluarga (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kartu_keluarga_id   UUID,
    nik                 VARCHAR(16) NOT NULL UNIQUE,
    nama_lengkap        VARCHAR(150) NOT NULL,
    tempat_lahir        VARCHAR(100) NOT NULL,
    tanggal_lahir       DATE NOT NULL,
    jenis_kelamin       VARCHAR(1) NOT NULL CHECK (jenis_kelamin IN ('L', 'P')),
    agama               VARCHAR(30) NOT NULL,
    pekerjaan           VARCHAR(100),
    nomor_hp            VARCHAR(20),
    hubungan_keluarga   VARCHAR(50) NOT NULL,
    status              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMP
);

CREATE TABLE IF NOT EXISTS kartu_keluarga (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nomor_kk             VARCHAR(16) NOT NULL UNIQUE,
    kepala_keluarga_id   UUID REFERENCES anggota_keluarga(id),
    nomor_hp             VARCHAR(20),
    alamat               TEXT NOT NULL,
    rt                   VARCHAR(5) NOT NULL,
    rw                   VARCHAR(5) NOT NULL,
    desa                 VARCHAR(100) NOT NULL,
    kecamatan            VARCHAR(100) NOT NULL,
    kabupaten            VARCHAR(100) NOT NULL,
    provinsi             VARCHAR(100) NOT NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMP
);

ALTER TABLE anggota_keluarga
    ADD CONSTRAINT fk_anggota_kk FOREIGN KEY (kartu_keluarga_id)
    REFERENCES kartu_keluarga(id) ON DELETE SET NULL;
