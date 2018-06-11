
module.exports = async function(context) {
    var status
    var randNumber = Math.floor(Math.random() * Math.floor(2))
    if ((randNumber%2) == 0) {
        status = 400
    } else {
        status = 200
    }
    console.log("status: " + status)
    console.log("random number: " + randNumber)
    return {
        status: status,
        body: "Bye!\n"
    };
}
