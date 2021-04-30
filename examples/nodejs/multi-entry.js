module.exports.entry1 = async (context) => {
    return {
        status: 200,
        body: "Hello, entry 1!\n"
    };
}

module.exports.entry2 = async (context) => {
    return {
        status: 200,
        body: "Hello, entry 2!\n"
    };
}
