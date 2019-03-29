FROM php:7.3-alpine

ENV PATH="/root/.composer/vendor/bin:${PATH}" \
    COMPOSER_ALLOW_SUPERUSER=1

RUN echo 'http://dl-4.alpinelinux.org/alpine/edge/testing' >> /etc/apk/repositories \
    && apk --no-cache add \
    git \
    curl \
    curl-dev \
    libcurl \
    zlib-dev \
    freetype-dev \
    jpeg-dev \
    libjpeg-turbo-dev \
    postgresql-dev \
    libmcrypt-dev \
    libpng-dev \
    icu-dev \
    gettext-dev \
    vim \
    libxml2-dev \
    freetype-dev \
    libzip-dev \
    unzip \
    libc6-compat \
    openssl \
    gcc \
    autoconf \
    make \
    libc-dev

RUN docker-php-ext-configure gd --with-freetype-dir=/usr/include/ --with-jpeg-dir=/usr/include/

# Install useful extensions
RUN docker-php-ext-install \
    opcache \
    bcmath \
    ctype \
    curl \
    dom \
    iconv \
    fileinfo \
    gd \
    gettext \
    intl \
    json \
    mysqli \
    pgsql \
    pcntl \
    pdo \
    ftp \
    pdo_mysql \
    pdo_pgsql \
    phar \
    simplexml \
    xmlrpc \
    zip

RUN pecl install \
    mcrypt-1.0.2
RUN docker-php-ext-enable \
    mcrypt

RUN curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer

COPY . /app
WORKDIR /app

RUN composer install

EXPOSE 8888

ENTRYPOINT ["php"]
CMD ["server.php"]
