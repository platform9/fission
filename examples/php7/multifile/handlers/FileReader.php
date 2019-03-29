<?php

use Psr\Http\Message\ResponseInterface;
use Psr\Log\LoggerInterface;

require __DIR__ . '/../vendor/autoload.php';

function execute($context)
{
    /** @var ResponseInterface $response */
    $response = $context["response"];
    /** @var LoggerInterface $logger */
    $logger = $context["logger"];
    $response->getBody()->write(file_get_contents(__DIR__ . '/message.txt'));
    $logger->debug('File read: example.txt');
}
