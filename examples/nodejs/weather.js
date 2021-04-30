'use strict';

const fetch = require('node-fetch');

module.exports = async (context) => {
    const location = ctx.request.body.location;

    if (!location) {
        return {
            status: 400,
            body: {
                text: 'You must provide a location.'
            }
        };
    }

    try {
        const locationSearch = await fetch(`https://www.metaweather.com/api/location/search/?query=${encodeURIComponent(location)}`).then(res => JSON.parse(res));
        const weatherResponse = await fetch(`https://www.metaweather.com/api/location/(${locationSearch[0].woeid}/`).then(res => JSON.parse(res));
        const weather = weatherResponse.consolidatedWeather[0]
        return {
            status: 200,
            body: {
                text: `It is ${weather.the_temp} celsius degrees in ${location} and ${weather.weather_state_name}`
            },
            headers: {
                'Content-Type': 'application/json'
            }
        };
    } catch (e) {
        console.error(e);
        return {
            status: 500,
            body: e
        };
    }
}