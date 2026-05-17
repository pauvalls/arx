from flask import Flask
from infrastructure.database import db_session
from domain.models import Order

app = Flask(__name__)

@app.route('/orders')
def list_orders():
    return db_session.query(Order).all()
