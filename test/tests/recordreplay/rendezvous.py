from flask import request

def main():
    time = request.args.get('time')
    date = request.args.get('date')
    r = "We'll meet at {} on {}.\n".format(time, date)
    return r
