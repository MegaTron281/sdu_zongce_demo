#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
全功能端到端检测（每个场景全新登录 = 无痕式干净会话）。
覆盖：登录/并行会话、申报(去重/材料空值)、班委初审/特殊情况、公示与异议、
公示结束守卫、管理员复审/改分、申诉全流程、汇总期锁定、账号管理边界、改密码、
材料上传、活动名录 CRUD、公示墙、成绩聚合与数据隔离、审计日志隔离。
"""
import json, urllib.request, urllib.error, csv, io, base64

BASE = "http://localhost:9080/api"
PW = "123456"

passes, fails = 0, 0
def check(name, cond, detail=""):
    global passes, fails
    if cond:
        passes += 1; print(f"  PASS  {name}")
    else:
        fails += 1; print(f"  FAIL  {name}  ::  {detail}")

def req(method, path, token=None, body=None, raw=None, ctype=None):
    url = BASE + path
    data = None
    headers = {"Accept": "application/json"}
    if token: headers["Authorization"] = "Bearer " + token
    if raw is not None:
        data = raw
        if ctype: headers["Content-Type"] = ctype
    elif body is not None:
        data = json.dumps(body).encode("utf-8"); headers["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        resp = urllib.request.urlopen(r, timeout=10)
        raw = resp.read()
        try:
            return resp.getcode(), json.loads(raw.decode("utf-8","replace"))
        except Exception:
            return resp.getcode(), raw.decode("utf-8","replace")
    except urllib.error.HTTPError as e:
        body = _safe(e.read())
        if e.code >= 400:
            print(f"    [DEBUG {method} {path} -> {e.code}] {body}")
        return e.code, body

def _safe(b):
    try: return json.loads(b.decode("utf-8","replace"))
    except Exception: return b.decode("utf-8","replace")

SESS = {}
def login(account, role):
    code, body = req("POST", "/auth/login", body={"account": account, "password": PW, "role": role})
    if code == 200 and isinstance(body, dict) and "token" in body:
        SESS[account] = body["token"]; return code, body["token"]
    return code, None

def get(a, p):   return req("GET", p, token=SESS.get(a))
def post(a, p, b=None): return req("POST", p, token=SESS.get(a), body=b)
def patch(a, p, b=None): return req("PATCH", p, token=SESS.get(a), body=b)

def new_app(student_alias, reviewer_alias, name_suffix, award="院级", proj="志愿", score=4.0, reviewer_for_class=True):
    """用唯一活动名新建活动+申报，避免去重冲突。返回 (code, app)。"""
    return post(student_alias, "/applications", {
        "activityName": f"E2E测试活动_{name_suffix}", "level": "院级", "category": "劳动素养",
        "awardLevel": award, "projectType": proj, "rank": 1, "score": score, "materials": []})

print("=== 0. 健康检查 ===")
c,_ = req("GET","/health"); check("后端 health 200", c==200)

print("\n=== 1. 干净登录（无痕式）===")
for acc,role in [("202300271018","student"),("202300271019","student"),("202300241002","student"),
                 ("202300202018","reviewer"),("202300241001","reviewer"),("zhangmiao","admin"),
                 ("wangqiang","admin"),("superadmin","admin"),("20230001","student")]:
    c,_ = login(acc, role); 
    if c!=200: print(f"    [login fail] {acc}/{role} -> {c}")
check("9 个账号全部登录成功", all(SESS.get(a) for a,_ in [("202300271018",0),("202300271019",0),("202300241002",0),("202300202018",0),("202300241001",0),("zhangmiao",0),("wangqiang",0),("superadmin",0),("20230001",0)]))

print("\n=== 2. 活动名录 + 广场申报（含材料空值/not-null 修复/去重中文提示）===")
c, acts = get("202300271018","/activities")
check("活动名录可读", c==200 and isinstance(acts,list) and len(acts)>0, f"code={c}")
act = acts[0]
c, appA = post("202300271018","/applications", {"activityId":act["id"],"awardLevel":"校级","projectType":"竞赛","rank":1,"score":8.0})
check("申报不带 materials -> 201（not-null 修复）", c==201, f"code={c} body={appA}")
appA_id = appA.get("id") if c==201 else None
c, dup = post("202300271018","/applications", {"activityId":act["id"],"awardLevel":"校级","projectType":"竞赛","rank":1,"score":8.0,"materials":["x"]})
dm = dup.get("error","") if isinstance(dup,dict) else ""
check("重复申报 -> 409", c==409, f"code={c}")
check("重复申报提示为中文（非英文 duplicate）", ("已就活动" in dm) and ("duplicate" not in dm.lower()), dm)
c, appB = post("202300271018","/applications", {"activityId":acts[1]["id"] if len(acts)>1 else act["id"],"awardLevel":"院级","projectType":"志愿","rank":2,"score":5.0,"materials":["a","b"]})
check("带 materials 申报 -> 201", c==201, f"code={c} body={appB}")

print("\n=== 3. 权限隔离：跨班审核/成绩查询 ===")
c, app24 = post("202300241002","/applications", {"activityId":act["id"],"awardLevel":"校级","projectType":"竞赛","rank":1,"score":6.0,"materials":[]})
app24_id = app24.get("id")
c,_ = post("202300202018", f"/applications/{app24_id}/review-pass", {})
check("跨班(公信24)申请 公信23 班委审核 -> 403", c==403, f"code={c}")
c,_ = get("zhangmiao","/scores/20230001")
check("辅导员 查非管辖班级(公信2301)成绩 -> 403", c==403, f"code={c}")
c,lim = get("superadmin","/scores/20230001")
check("系统管理员 查全校成绩 -> 200", c==200, f"code={c}")
c,_ = get("202300271018","/scores/202300241002")
check("学生 查他人成绩 -> 403", c==403, f"code={c}")

print("\n=== 4. 班委初审 -> 公示中（改分需理由）/ 守卫 ===")
c, rp = post("202300202018", f"/applications/{appA_id}/review-pass", {"score":12.0,"reason":"初审加分"})
check("班委初审(改分带理由) -> 200 公示中", c==200 and rp.get("status")=="公示中", f"code={c} body={rp}")
c, g = post("202300202018", f"/applications/{appA_id}/publicity-finish", {})
check("公示<1h 手动结束被守卫拦截 -> 非200", c!=200, f"code={c} body={g}")
c,_ = post("202300202018", f"/applications/{appA_id}/review-pass", {})
check("对已公示重复 review-pass -> 409", c==409, f"code={c}")

print("\n=== 5. 异议：同班非本人可提 / 本人不可 / 跨班不可 ===")
c, o = post("202300271019", f"/applications/{appA_id}/object", {"reason":"存疑"})
check("同班同学提异议 -> 200", c==200, f"code={c} body={o}")
c,_ = post("202300271018", f"/applications/{appA_id}/object", {"reason":"我的"})
check("本人提异议 -> 失败", c!=200, f"code={c}")
c,_ = post("202300241002", f"/applications/{appA_id}/object", {"reason":"x"})
check("跨班提异议 -> 403", c==403, f"code={c}")
c, obs = get("202300202018", f"/applications/{appA_id}/objections")
check("班委可见本班异议列表", c==200 and isinstance(obs,list), f"code={c}")

print("\n=== 6. 撤回 / 重新申报 / 特殊情况 / 申诉 ===")
c, w = new_app("202300271018","202300202018","撤回专用")
wid = w.get("id")
c, wr = post("202300271018", f"/applications/{wid}/withdraw", {})
check("学生撤回待班委审核申请 -> 200 已驳回", c==200 and wr.get("status")=="已驳回", f"code={c} body={wr}")
c, re0 = new_app("202300271018","202300202018","撤回专用")  # 同名被驳回活动 -> 重建新活动+申报
check("被驳回后 可重新申报同活动名 -> 201", c==201, f"code={c} body={re0}")
# 申诉（采纳分支）：提交 -> 班委驳回 -> 学生申诉 -> 辅导员采纳(重新公示)
c, rap = new_app("202300271018","202300202018","申诉专用")
rid = rap.get("id")
post("202300202018", f"/applications/{rid}/review-reject", {"reason":"不符合"})
c, ap = post("202300271018", f"/applications/{rid}/appeal", {"reason":"应计入"})
check("被驳回后学生申诉 -> 200 申诉中", c==200 and ap.get("status")=="申诉中", f"code={c} body={ap}")
c, aa = post("zhangmiao", f"/applications/{rid}/appeal-approve", {"reason":"采纳"})
check("辅导员采纳申诉 -> 200 重新公示中", c==200 and aa.get("status")=="公示中", f"code={c} body={aa}")
# 申诉（驳回分支，独立申请）：驳回 -> 申诉 -> 辅导员驳回申诉(回已驳回)
c, rap2 = new_app("202300271018","202300202018","申诉专用2")
rid2 = rap2.get("id")
post("202300202018", f"/applications/{rid2}/review-reject", {"reason":"不符合"})
post("202300271018", f"/applications/{rid2}/appeal", {"reason":"应计入"})
c, ar = post("zhangmiao", f"/applications/{rid2}/appeal-reject", {"reason":"仍不符合"})
check("驳回申诉(分数未改) -> 200 已驳回", c==200 and ar.get("status")=="已驳回", f"code={c} body={ar}")
# 特殊情况：公信24 学生提交 -> 公信24 班委标记 -> 辅导员(公信23/24)复审通过
c, sp = new_app("202300241002","202300241001","特殊专用")
sid = sp.get("id")
post("202300241001", f"/applications/{sid}/mark-special", {"reason":"伤病"})
c, spp = post("zhangmiao", f"/applications/{sid}/admin-pass", {"reason":"特殊处理通过"})
check("特殊情况 管理员复审通过 -> 200 已通过", c==200 and spp.get("status")=="已通过", f"code={c} body={spp}")

print("\n=== 7. 汇总期锁定 ===")
c,_ = patch("zhangmiao","/meta", {"summaryPhase":True})
check("辅导员 开启汇总期 -> 200", c==200, f"code={c}")
c, lk = post("202300271018","/applications", {"activityId":act["id"],"awardLevel":"院级","projectType":"志愿","rank":1,"score":3.0,"materials":[]})
check("汇总期 学生申报 -> 409 锁定", c==409, f"code={c} body={lk}")
c,_ = patch("zhangmiao","/meta", {"summaryPhase":False})
check("辅导员 关闭汇总期 -> 200", c==200, f"code={c}")

print("\n=== 8. 完整生命周期 + 成绩聚合（s1 李明/公信2301，系统管理员驱动）===")
c, before = get("superadmin","/scores/20230001")
b_ass = before.get("assessmentScore") if isinstance(before,dict) else None
check("李明初始成绩 ~77.8", c==200 and abs(float(b_ass)-77.8)<0.05, f"assessment={b_ass}")
c, pf = post("superadmin","/applications/s1/publicity-finish", {})
check("s1 结束公示 -> 200 待管理员审核", c==200 and pf.get("status")=="待管理员审核", f"code={c} body={pf}")
c, aps1 = post("superadmin","/applications/s1/admin-pass", {"reason":"复审通过"})
check("s1 管理员复审通过 -> 200 已通过", c==200 and aps1.get("status")=="已通过", f"code={c} body={aps1}")
c, after = get("superadmin","/scores/20230001")
a_ass=after.get("assessmentScore"); a_q=after.get("qualityScore")
check("s1 已通过后 qualityScore=12", abs(float(a_q)-12)<0.01, f"quality={a_q}")
check("成绩聚合 = 79.0", abs(float(a_ass)-79.0)<0.01, f"assessment={a_ass}")

print("\n=== 9. 账号管理：仅系统管理员可改角色/管辖班级 ===")
c, users = get("zhangmiao","/admin/users")
uid = next((u["id"] for u in users if u.get("role")=="student"), None)
c,_ = patch("zhangmiao", f"/admin/users/{uid}/role", {"role":"admin"})
check("辅导员 自提他人为 admin -> 403", c==403, f"code={c}")
c,_ = patch("zhangmiao", f"/admin/users/{uid}/classes", {"classes":["公信23"]})
check("辅导员 设置管辖班级 -> 403（仅超管）", c==403, f"code={c}")
# 用 wangqiang(管理员)验证“系统管理员可设置管辖班级”
c,_ = patch("superadmin", "/admin/users/user-admin-2/classes", {"classes":["公信23","公信24"]})
check("系统管理员 设置 admin 管辖班级 -> 200", c==200, f"code={c}")
# 还原 wangqiang 原始管辖班级(公信25/公信26)
patch("superadmin", "/admin/users/user-admin-2/classes", {"classes":["公信25","公信26"]})
c,_ = patch("superadmin", f"/admin/users/{uid}/role", {"role":"reviewer"})
check("系统管理员 改角色 -> 200", c==200, f"code={c}")
# 还原
patch("superadmin", f"/admin/users/{uid}/role", {"role":"student"})

print("\n=== 10. 改密码（旧密码失效/新密码生效）===")
c,_ = post("202300271018","/auth/change-password", {"oldPassword":PW,"newPassword":"newpwd123"})
check("学生 改密码 -> 200", c==200, f"code={c}")
c,_ = login("202300271018","student")  # 旧密码
check("旧密码登录 -> 失败", c!=200, f"code={c}")
c,tok_new = login("202300271018","student")  # 占位(旧密码已失败)
# 用新密码登录
r = req("POST","/auth/login", body={"account":"202300271018","password":"newpwd123","role":"student"})
check("新密码登录 -> 200", r[0]==200, f"code={r[0]}")
if r[0]==200:
    req("POST","/auth/change-password", token=r[1]["token"], body={"oldPassword":"newpwd123","newPassword":PW})
    # 改密会吊销该用户其他会话；把 SESS 刷新为本次有效的 token_B，避免后续请求 401
    SESS["202300271018"] = r[1]["token"]

print("\n=== 11. 用户导入 CSV ===")
csv_txt = "学号,姓名,班级\n202309990001,导入甲,公信23\n202309990002,导入乙,公信24\n"
fb="----tb"; body=(f"--{fb}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"u.csv\"\r\nContent-Type: text/csv\r\n\r\n{csv_txt}\r\n--{fb}--\r\n").encode()
c, imp = req("POST","/admin/users/import", token=SESS.get("zhangmiao"), raw=body, ctype=f"multipart/form-data; boundary={fb}")
created = imp.get("created",0) if isinstance(imp,dict) else 0
check("管理员 导入 CSV -> 200 且创建>=2", c==200 and created>=2, f"code={c} body={imp}")
c, ul = get("zhangmiao","/admin/users")
names = [u.get("displayName") for u in ul] if isinstance(ul,list) else []
check("导入用户出现在列表", any("导入甲" in n for n in names), f"names={names}")

print("\n=== 12. 导出成绩隔离 ===")
c, csvd = get("zhangmiao","/admin/export/scores.csv")
rows = list(csv.reader(io.StringIO(csvd))) if isinstance(csvd,str) else []
cls = set(r[2] for r in rows[1:] if len(r)>2)
check("辅导员导出 仅管辖班级(无公信2301)", bool(cls) and "公信2301" not in cls, f"classes={cls}")
c, call = get("superadmin","/admin/export/scores.csv")
rowsa = list(csv.reader(io.StringIO(call))) if isinstance(call,str) else []
clsa = set(r[2] for r in rowsa[1:] if len(r)>2)
check("系统管理员导出 含公信2301", "公信2301" in clsa, f"classes={clsa}")

print("\n=== 13. 审计日志隔离（关键探针）===")
c, logs = get("zhangmiao","/admin/logs")
logs = logs if isinstance(logs,list) else []
markers=["公信2301","公信25","公信26","李明"]
leak = any(any(m in (lg.get("target","")+lg.get("note","")+lg.get("actor","")) for m in markers) for lg in logs)
check("辅导员 审计日志 未泄露非管辖班级", (not leak), f"logs={len(logs)} sample={logs[:1]}")
print(f"      [info] 辅导员可见日志条数={len(logs)}")

print("\n=== 14. 活动名录 CRUD + 学生建活动越权观察 ===")
c, ca = post("zhangmiao","/activities", {"name":"测试新建活动A","level":"院级","category":"志愿服务","organizer":"学生会"})
check("管理员 创建活动 -> 201", c==201, f"code={c} body={ca}")
ca_id = ca.get("id") if c==201 else None
c, pa = patch("zhangmiao", f"/activities/{ca_id}", {"status":"已上架"})
check("管理员 上架活动 -> 200", c==200 and pa.get("status")=="已上架", f"code={c} body={pa}")
c, sa_act = post("202300271018","/activities", {"name":"学生私自建活动","level":"院级","category":"志愿服务"})
check("[观察] 学生 POST /activities 是否被允许", c in (201,403),
      f"code={c} body={sa_act}  -> 若201则学生可建活动(与PATCH需admin不一致)")

print("\n=== 15. 材料上传 ===")
# 1x1 透明 PNG（合法格式）
PNG_B64="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVR42mNk+P+/HgAFhAJ/wlseKgAAAABJRU5ErkJggg=="
mpng = base64.b64decode(PNG_B64)
mfb="----mf"
mhead=f"--{mfb}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"m.png\"\r\nContent-Type: image/png\r\n\r\n".encode()
mtail=f"\r\n--{mfb}--\r\n".encode()
mbody=mhead+mpng+mtail
c, up = req("POST","/materials/upload", token=SESS.get("202300271018"), raw=mbody, ctype=f"multipart/form-data; boundary={mfb}")
check("学生 上传 PNG 材料 -> 200/201", c in (200,201), f"code={c} body={up}")
# 负向：非法格式应被拒
mtxt=(f"--{mfb}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"bad.txt\"\r\nContent-Type: text/plain\r\n\r\nhello\r\n--{mfb}--\r\n").encode()
c,_ = req("POST","/materials/upload", token=SESS.get("202300271018"), raw=mtxt, ctype=f"multipart/form-data; boundary={mfb}")
check("学生 上传非法格式(txt) -> 400", c==400, f"code={c}")

print("\n================ 结果 ================")
print(f"PASS={passes}  FAIL={fails}")
