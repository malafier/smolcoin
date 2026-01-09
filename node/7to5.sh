#!/bin/bash

curl -X POST http://127.0.0.1:3007/admin/connect \
     -H "X-Admin-Key: secret" \
     -H "Content-Type: application/json" \
     -d '{"peer":"localhost:3005"}'
