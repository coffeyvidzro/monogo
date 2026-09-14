variable "database_url" {
  type    = string
  default = "postgres://postgres:postgres@localhost:5432/leamout?sslmode=disable&search_path=public"
}

variable "dev_database_url" {
  type    = string
  default = "postgres://postgres:postgres@localhost:5432/leamout_atlas_dev?sslmode=disable"
}

env "local" {
  url = var.database_url
  dev = var.dev_database_url

  migration {
    dir = "file://migrations"
  }

  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}