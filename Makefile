current_dir = $(shell pwd)

plan:
	@terraform plan

deploy:
	@terraform apply