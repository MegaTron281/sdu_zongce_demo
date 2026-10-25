import json, urllib.request, urllib.error, random, string

BASE = "http://localhost:9080/api"
PASS, FAIL = 0, 0

def rnd_suffix(n=6):
    return "".join(random.choices(string.digits, k=n))

def req(method, path, token=None, body=None):
    url = BASE + path
    data = None
    headers = {"Accept": "application/json", "Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    if body is not None:
        data = json.dumps(body).encode()
    r = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        resp = urllib.request.urlopen(r, timeout=10)
        code = resp.getcode()
        try:
            return code, json.loads(resp.read().decode())
        except Exception:
            return code, resp.read().decode()
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read().decode())
        except Exception:
            return e.code, e.read().decode()

def login(account, role, pwd="123456"):
    c, b = req("POST", "/auth/login", body={"account": account, "password": pwd, "role": role})
    if c == 200 and isinstance(b, dict):
        return c, b.get("token")
    return c, None

def check(name, cond, detail=""):
    global PASS, FAIL
    if cond:
        PASS += 1
        print(f"  [PASS] {name}")
    else:
        FAIL += 1
        print(f"  [FAIL] {name}  {detail}")

print("=== 优化2：活动名录权限统一（POST 仅 admin）===")
# 学生登录
c_stu, tok_stu = login("20230001", "student")
check("学生登录 -> 200", c_stu == 200, f"code={c_stu}")
# 学生 POST 活动 -> 应 403
c, b = req("POST", "/activities", token=tok_stu, body={"name": "学生越权建活动测试", "category": "文艺素养"})
check("学生 POST /activities -> 403", c == 403, f"code={c} body={b}")
# 学生仍能 GET 活动广场
c, b = req("GET", "/activities", token=tok_stu)
check("学生 GET /activities -> 200", c == 200, f"code={c}")
# 辅导员登录
c_adm, tok_adm = login("zhangmiao", "admin")
check("辅导员登录 -> 200", c_adm == 200, f"code={c_adm}")
# admin POST 活动 -> 应 201（唯一名称避免重名误报）
act_name = "优化验证_管理建活动_" + rnd_suffix()
c, b = req("POST", "/activities", token=tok_adm, body={"name": act_name, "category": "文艺素养", "organizer": "校团委"})
check("admin POST /activities -> 201", c == 201, f"code={c} body={b}")
if c == 201 and isinstance(b, dict):
    act_id = b.get("id")
    # admin PATCH 仍可
    c2, _ = req("PATCH", f"/activities/{act_id}", token=tok_adm, body={"status": "已上架"})
    check("admin PATCH /activities/:id -> 200", c2 == 200, f"code={c2}")
    # 学生 PATCH -> 403（不变）
    c3, _ = req("PATCH", f"/activities/{act_id}", token=tok_stu, body={"status": "已上架"})
    check("学生 PATCH /activities/:id -> 403", c3 == 403, f"code={c3}")
    # 清理
    req("PATCH", f"/activities/{act_id}", token=tok_adm, body={"status": "待审核上架"})

print("\n=== 优化3：改密后吊销其他会话（保留当前）===")
# 同一学生开两个会话
cA, tokA = login("20230003", "student")
cB, tokB = login("20230003", "student")
check("账号 20230003 会话A 登录 -> 200", cA == 200, f"code={cA}")
check("账号 20230003 会话B 登录 -> 200", cB == 200, f"code={cB}")
# 改密（用会话A令牌），旧密码 123456 -> 新密码 newpwd1
c, b = req("POST", "/auth/change-password", token=tokA, body={"oldPassword": "123456", "newPassword": "newpwd1"})
check("会话A 改密 -> 200", c == 200, f"code={c} body={b}")
# 当前会话(A) 仍有效
c, b = req("GET", "/auth/me", token=tokA)
check("改密后 当前会话A 仍有效(200)", c == 200, f"code={c}")
# 其他会话(B) 被吊销
c, b = req("GET", "/auth/me", token=tokB)
check("改密后 其他会话B 被吊销(401)", c == 401, f"code={c} body={b}")
# 用旧密码登录应失败（密码已变，错误密码返回 400）
c, b = login("20230003", "student", pwd="123456")
check("旧密码登录 -> 400（密码已变更）", c == 400, f"code={c}")
# 用新密码登录应成功
c, tokNew = login("20230003", "student", pwd="newpwd1")
check("新密码登录 -> 200", c == 200, f"code={c}")
# 还原密码为 123456（用当前有效会话 tokNew）
c, b = req("POST", "/auth/change-password", token=tokNew, body={"oldPassword": "newpwd1", "newPassword": "123456"})
check("还原密码 -> 200", c == 200, f"code={c} body={b}")
# 还原后新密码登录成功
c, _ = login("20230003", "student", pwd="123456")
check("还原后 123456 登录 -> 200", c == 200, f"code={c}")

print(f"\n=== 结果：PASS={PASS}  FAIL={FAIL} ===")
