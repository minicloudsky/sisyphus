# Sisyphus Quick Start

**Languages:** **English** (this page) | [简体中文](QUICKSTART.zh-CN.md)

---

## 1. Environment

### MySQL & Redis (example: macOS + Homebrew)

```bash
brew services list | grep mysql
brew services start mysql

redis-cli ping
brew services start redis
```

---

## 2. Initialize the database

### Option A: setup script (recommended)

```bash
cd sisyphus
./scripts/setup.sh
```

You will be prompted for the MySQL `root` password. The script creates the database, schema, test data, and updates config where applicable.

### Option B: manual

```bash
mysql -u root -p -e "CREATE DATABASE sisyphus DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p sisyphus < migrations/init.sql
mysql -u root -p sisyphus < migrations/test_data.sql
```

If your repo layout still references legacy `sql/` paths, use the files under `migrations/` instead.

---

## 3. Configure the backend

Edit `server/configs/config.yaml` (or copy to `config.local.yaml`):

```yaml
data:
  database:
    source: root:YOUR_PASSWORD@tcp(127.0.0.1:3306)/sisyphus?charset=utf8mb4&parseTime=True&loc=Local
```

Empty password example:

```yaml
data:
  database:
    source: root@tcp(127.0.0.1:3306)/sisyphus?charset=utf8mb4&parseTime=True&loc=Local
```

Also set Redis, `auth.jwt_secret`, and **goInception** host/port.

---

## 4. Run the backend

```bash
cd server
go mod download
make wire    # if Wire codegen is needed
go run cmd/sisyphus/main.go -conf configs/config.yaml
```

Or use project scripts / Makefile targets if documented in `server/README.md`.

Backend default: `http://localhost:8000`.

---

## 5. Run the frontend

```bash
cd web
npm install
npm start
```

Frontend default: `http://localhost:8001`.

---

## 6. Sign in

| User | Password |
|------|----------|
| `admin` | `Admin@123` |
| `dba` | `Admin@123` |
| `developer` | `Admin@123` |

---

## 7. Optional: goInception

SQL order review requires goInception listening (often port `4000`). Install from [goInception releases](https://github.com/hanchuanchuan/goInception/releases) and start with your `inception.cnf`.

---

## Next steps

- Full overview: [README.md](README.md)  
- Feature walkthrough: [docs/GUIDE.md](docs/GUIDE.md)  
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
