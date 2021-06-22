FROM php:7.1-fpm

RUN apt-get update \
    && docker-php-ext-install mysqli pdo_mysql && docker-php-ext-enable mysqli pdo_mysql
