module scroll-tech/store

go 1.17

require (
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.6
	github.com/mattn/go-sqlite3 v1.14.12
	github.com/pressly/goose/v3 v3.7.0
	github.com/scroll-tech/go-ethereum v1.10.14-0.20220920070544-3a7da33cd53d
	github.com/stretchr/testify v1.7.2
	scroll-tech/internal v1.0.0
)

replace (
	scroll-tech/bridge v1.0.0 => ../bridge
	scroll-tech/coordinator v1.0.0 => ../coordinator
	scroll-tech/internal v1.0.0 => ../internal
)

require (
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker v20.10.17+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/ethereum/go-ethereum v1.10.14 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/iden3/go-iden3-crypto v0.0.12 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa // indirect
	golang.org/x/net v0.0.0-20220812174116-3211cb980234 // indirect
	golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
