# Go-Ecommerce
Go-Ecommerce is a sample ecommerce microservice written in go in a mono repo for the teaching how to create microservices in golang.

## Lesson 1: Repo Setup
- Create a git repository called go-ecommerce
- Create a directory on you computer and name it `go-ecommerce`
- Run the command: `cd <path-to-go-ecommerce>`
- Run the command: `git init`
- Run the command: `git remote add origin <url-of-the-git-repo>`
- Add .gitignore to the local repo

## Lesson 2: Products REST API Setup
- Add a subdirectory to the `go-ecommerce` folder and name it `products`
- Add `golangci.yml` file to the `products` directory root
- Add `Makefile` file to the products directory root
- Create the directories `./products/cmd/api`
- CD into `products` directory
- Run go mod init:
```
go mod init <repo/go-ecommerce/products-path>
```