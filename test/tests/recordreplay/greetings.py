from flask import request

def main():
    content = request.get_json(force=True)
    title = content['title']
    name = content['name']
    item = content['item']
    g = "Greetings, {} {}. May I take your {}?\n".format(title, name, item)
    return g
