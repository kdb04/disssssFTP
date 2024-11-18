import math


def idf_func(word, n):
    count = 0
    for doc in range(n):
        words = list(map[doc].keys())
        if word in words:
            count += 1
    print("IDF(" + word + ") = " + str(math.log(n/count)))  


map = []
n = int(input("Input the number of documents: "))
for i in range(n):
    doc = input("Enter the contents of document " + str(i + 1) + " : ")
    doc = doc.split(" ")
    map.append({})
    for word in doc:
        if word not in map[i]:
            map[i][word] = 1
        else:
            map[i][word] += 1

for i in range(n):
    keys_list = list(map[i].keys())
    for j in range(len(map[i])):
        print("TF(" + keys_list[j] + ") : " + str(map[i][keys_list[j]]/len(map[i])))

print("\n")
idf = []
for i in range(n):
    keys_list = list(map[i].keys())
    for j in range(len(map[i])):
        if keys_list[j] not in idf:
            idf.append(keys_list[j])
            idf_func(keys_list[j], n)
        else:
            continue
