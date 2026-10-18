current_dir = $(shell pwd)

# Terraform root to run: deploy/gke (default), deploy/aks or deploy/eks,
# e.g. make plan TF_DIR=deploy/eks
TF_DIR ?= deploy/gke
# AKS and EKS keep their state settings in a file, e.g. BACKEND_CONFIG=backend.hcl
BACKEND_CONFIG ?=

init:
	@terraform -chdir=$(TF_DIR) init $(if $(BACKEND_CONFIG),-backend-config=$(BACKEND_CONFIG))

plan:
	@terraform -chdir=$(TF_DIR) plan

deploy:
	@terraform -chdir=$(TF_DIR) apply

deploy-auto:
	@terraform -chdir=$(TF_DIR) apply --auto-approve
