current_dir = $(shell pwd)

mysql-clean:
	docker rm -f mrssanta-mysql || true
	rm -rf mysql

mysql:
	docker rm -f mrssanta-mysql || true
	docker run --name mrssanta-mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=mrssanta -e MYSQL_USER=mrssanta -e MYSQL_PASSWORD=password -v ${current_dir}/mysql:/var/lib/postgresql/data -d mysql:8