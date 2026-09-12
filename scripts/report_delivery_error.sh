#!/bin/sh
set -eu

curl --fail-with-body --request POST http://localhost:8080/delivery-errors \
  --header 'Content-Type: application/json' \
  --data '{"course_id":"course-42","learner_id":"learner-7","delivery_stage":"lesson-publish","deadline":"2026-08-17T15:00:00Z","message":"lesson package could not be delivered","exception":"publish lesson: object validation failed"}'
