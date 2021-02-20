from PyQt5 import QtWidgets, uic
from sys import argv
from os import path
from logic import *

path_to_ui =path.join(path.dirname(__file__), 'main_window.ui')

settings = dict(format='-pdf', delete=False, url='')

class Ui(QtWidgets.QMainWindow):
    def __init__(self):
        super(Ui, self).__init__()
        uic.loadUi(path_to_ui, self) 
        self.show() 


app = QtWidgets.QApplication(argv)
window = Ui()

connect_buttons(window, settings)

app.exec_()
