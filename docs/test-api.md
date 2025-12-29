Test API Commands

# Create tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" -d '{"name":"my-tenant"}'

# Create credential (password base64 encoded)
curl -X POST http://localhost:8080/api/v1/credentials \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"...","kind":"source","plaintext":"dGVzdC1wYXNz"}'

# Create SSH source
curl -X POST http://localhost:8080/api/v1/sources \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"...","type":"ssh","name":"server","credential_id":"...","config":{"host":"10.0.100.85","port":22,"username":"web","paths":["/home/web/test"],"use_password":true}}'

# Enqueue backup job
curl -X POST "http://localhost:8080/api/v1/jobs?tenant_id=..." \
  -H "Content-Type: application/json" -d '{"source_id":"..."}'
Quick Reference
Endpoint	Method	Purpose
/api/v1/tenants	POST	Create tenant with keypair
/api/v1/credentials	POST	Create encrypted credential
/api/v1/sources	POST	Create backup source
/api/v1/sources	GET	List sources
/api/v1/jobs	POST	Enqueue backup job
/api/v1/snapshots	GET	List snapshots
/internal/jobs/claim	POST	Worker claims job
/internal/jobs/:id/complete	POST	Worker reports completion
