t = {a = 1, b = 2, c = 3}
for k, v in pairs(t) do
  print(k, v)
end

t = {"a", "b", "c"}
for i, v in ipairs(t) do
  print(i, v)
end