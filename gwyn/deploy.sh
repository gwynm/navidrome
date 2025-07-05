#!/bin/bash
set -e # exit on error

aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws


DOCKER_TAG=gwynm/navidrome:develop IMAGE_PLATFORMS=linux/amd64 make docker-image
docker push gwynm/navidrome:develop
ssh nuc@nuc.fritz.box 'cd ~/Documents/docker && docker compose up -d navidrome'

