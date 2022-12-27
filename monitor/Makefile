DEPLOY_DIR = ../ratel/gauge/www

.PHONY: deploy
deploy:
	npm run build
	rm -rf ${DEPLOY_DIR}/*
	cp -r build/* ${DEPLOY_DIR}/
