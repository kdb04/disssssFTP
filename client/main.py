from typing import Optional
from fastapi import FastAPI, Path
from pydantic import BaseModel

app = FastAPI()

students = {
    1: {
        "name": "kk",
        "age": "20",
        "year": "enginej"
    }
}

class Student(BaseModel):
    name: str
    age: int
    year: str

class UpdateStudent(BaseModel):
    name: Optional[str] = None
    age: Optional[int] = None
    year: Optional[str] = None

@app.get("/")
def index():
    return {"name": "kk"}

@app.get("/get-student/{student_id}")
def get_student(student_id: int = Path(description="hiii u need to put student_id", gt=0, lt=4)):
    # greater than 0 less than 3
    return students.get(student_id)

@app.get("/get-by-name") 
def get_student_by_name(*, name: Optional[str] = None, test: int):
    for student_id in students:
        if students[student_id]["name"] == name:
            return students[student_id]
    return {"Data": "Notfound"}

@app.post("/create-student/{student_id}") # here student has to give all values to insert
def create_student(student_id: int, student: Student):
    if student_id in students:
        return {"Error": "Student existss"}

    students[student_id] = student
    return students[student_id]

@app.put("/update-student/{student_id}")
def update_student(student_id: int, student: UpdateStudent):
    if student_id not in students:
        return {"Error": "Cannot Update smtg which doesnt exist"}
    
    if student.name != None:
        students[student_id].name = student.name
    if student.age != None:
        students[student_id].age = student.age
    if student.year != None:
        students[student_id].year = student.year

    return students[student_id]

@app.delete("/delete-student/{student_id}")
def delete_student(student_id: int):
    if student_id not in students:
        return {"Error": "Cannot Delete smtg which doesnt exist"}
    
    del students[student_id]
    return {"SUccess": "DELETEDD"}

    

