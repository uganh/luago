# Implementation of Lua in Go Language

| C                           | Go        | size  |
| :---                        | :---      | :---: |
| `lu_Byte` (`unsigned char`) | `byte`    | 1     |
| `int`                       | `uint32`  | 4     |
| `size_t`                    | `uint64`  | 8     |
| `lu_Integer` (`long long`)  | `int64`   | 8     |
| `lu_Number` (`double`)      | `float64` | 8     |

## Instruction Set

+ 常量加载指令
+ 运算符相关指令
+ 循环和跳转指令
+ 函数调用相关指令
+ 表操作指令
+ Upvalue 操作指令

四种编码模式（Mode）：

```c
struct iABC {
  uint32_t opcode : 6;
  uint32_t A : 8;
  uint32_t B : 9;
  uint32_t C : 9;
};

struct iABx {
  uint32_t opcode : 6;
  uint32_t A : 8;
  uint32_t Bx : 18;
};

struct iAsBx {
  uint32_t opcode : 6;
  uint32_t A : 8;
  uint32_t sBx : 18; // signed
};

struct iAx {
  uint32_t opcode : 6;
  uint32_t Ax : 26;
};
```

## Lua API

### Data Types

+ `nil`
+ `boolean`
+ `number`
+ `string`
+ `table`
+ `function`
+ `thread`
+ `userdata`

| Lua       | Go        |
| :---      | :---      |
| `nil`     | `nil`     |
| `boolean` | `bool`    |
| `integer` | `int64`   |
| `float`   | `float64` |
| `string`  | `string`  |

### Stack

`0 < top <= cap`

+ Valid index: $[1, \text{top}]$
+ Acceptable index: $[1, \text{cap}]$
