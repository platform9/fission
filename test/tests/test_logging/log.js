
module.exports = async function(context) {
    console.log("log test log test log test")
    return {
        status: 200,
        body: "Log, test!\n"
    };
}
