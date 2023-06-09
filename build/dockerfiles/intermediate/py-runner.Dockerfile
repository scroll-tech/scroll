ARG PYTHON_VERSION=3.10
FROM python:${PYTHON_VERSION}-alpine

RUN apk add --no-cache gcc g++ make musl-dev
