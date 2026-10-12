#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""端到端功能检测：覆盖申报 bug 修复、完整审核生命周期、成绩聚合与权限隔离。"""
import json
import os
import sys
import urllib.request
import urllib.error

BASE = "http://localhost:8090/api"

# 1x1 透明 PNG（用于材料上传测试）
PNG_B64 = bytes.fromhex(
    "89504e470d0a1a0a0000000d49484452000000010000000108060000001f15c4"
    "890000000a49444154789c6360000002000001e221bc330000000049454e44ae426082"
)


def req(method, path, token=None, body=None, raw=None, files=None, ctype=None):
    url = BASE + path
    data = None
    headers = {}
    if token:
        headers["Authorization"] = "Bearer " + token
    if files is not None:
        import io

        boundary = "----e2eboundary"
        buf = io.BytesIO()
        for fname, fcontent in files:
            buf.write(("--" + boundary + "\r\n").encode())
            buf.write(
                ('Content-Disposition: form-data; name="file"; filename="%s"\r\n' % fname).encode()
            )
            buf.write(b"Content-Type: application/octet-stream\r\n\r\n")
            buf.write(fcontent)
            buf.write(b"\r\n")
        buf.write(("--" + boundary + "--\r\n").encode())
        data = buf.getvalue()
        headers["Content-Type"] = "multipart/form-data; boundary=" + boundary
    elif raw is not None:
        data = raw
        if ctype:
            headers["Content-Type"] = ctype
    elif body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(r, timeout=10) as resp:
            code = resp.getcode()
            text = resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        code = e.code
        text = e.read().decode("utf-8", "replace")
    try:
        payload = json.loads(text) if text else None
    except Exception:
        payload = text
    return code, payload


PASS = []
FAIL = []


def check(name, cond, detail=""):
    if cond:
        PASS.append(name)
        print("  [PASS] %s%s" % (name, ("  -> " + detail) if detail else ""))
    else:
        FAIL.append(name)
        print("  [FAIL] %s%s" % (name, ("  -> " + detail) if detail else ""))


def login(account, password, role):
    code, body = req("POST", "/auth/login", body={"account": account, "password": password, "role": role})
    assert code == 200, "login %s failed %d %s" % (account, code, body)
    return body["token"], body.get("user", {})


print("=== 0. 健康检查 ===")
code, body = req("GET", "/health")
check("后端 health 200", code == 200, str(body))

print("\n=== 1. 三角色登录（并行登录 / Bearer 令牌）===")
tok_stu, stu = login("202300221077", "123456", "student")   # 晏陆君 公信23（无历史申请）
tok_rev, rev = login("202300202018", "123456", "reviewer")  # 班委巩浩 公信23
tok_adm, adm = login("zhangmiao", "123456", "admin")        # 辅导员张苗（已授权管辖 公信2301）
tok_lim, lim = login("20230001", "123456", "student")       # 李明 公信2301
check("学生/班委/管理员/李明 均登录成功", all([tok_stu, tok_rev, tok_adm, tok_lim]))
code, meta = req("GET", "/meta", token=tok_stu)
check("meta 需登录且可读", code == 200 and isinstance(meta, dict) and "applicationStatuses" in meta, str(meta)[:80])

print("\n=== 2. 材料上传（multipart，修复 forbidden 问题）===")
code, up = req("POST", "/materials/upload", token=tok_stu, files=[("cert.png", PNG_B64)])
check("学生上传材料 201", code == 201, str(up))
mat_name = up.get("name") if isinstance(up, dict) else None
mat_url = up.get("url") if isinstance(up, dict) else None

print("\n=== 3. 申报流程（bug 修复验证）===")
# 3a 不带 materials 字段 -> 之前报 400 not-null，现在应 201
code, b = req("POST", "/applications", token=tok_stu, body={
    "activityId": "a4", "awardLevel": "二等奖", "projectType": "个人项目", "rank": 1, "score": 6})
check("无 materials 字段申报 -> 201（not-null bug 已修复）", code == 201, "appId=%s" % (b.get("id") if isinstance(b, dict) else b))
app_a4 = b.get("id") if isinstance(b, dict) else None

# 3b 重复申报同一活动 -> 409 且返回中文提示（核心 bug 修复）
code, b2 = req("POST", "/applications", token=tok_stu, body={
    "activityId": "a4", "awardLevel": "二等奖", "projectType": "个人项目", "rank": 1, "score": 6})
err_msg = b2.get("error", "") if isinstance(b2, dict) else str(b2)
check("重复申报 a4 -> 409", code == 409, "code=%d" % code)
check("重复申报提示为中文（无英文 duplicate active application）", ("重复" in err_msg or "已" in err_msg) and "duplicate" not in err_msg.lower(),
      err_msg[:60])

# 3c 带材料的申报 -> 201
code, b3 = req("POST", "/applications", token=tok_stu, body={
    "activityId": "a1", "awardLevel": "一等奖", "projectType": "个人项目", "rank": 1, "score": 12,
    "materials": [mat_url] if mat_url else ["cert.png"]})
check("带材料申报 a1 -> 201", code == 201, "appId=%s" % (b3.get("id") if isinstance(b3, dict) else b3))
app_a1 = b3.get("id") if isinstance(b3, dict) else None

print("\n=== 4. 权限隔离 / 错误路径 ===")
# 4a 学生不能查他人成绩 -> 403
code, _ = req("GET", "/scores/20230001", token=tok_stu)
check("学生查他人成绩 -> 403", code == 403, "code=%d" % code)
# 4b 学生不能执行管理员动作 -> 403
code, _ = req("POST", "/applications/s1/admin-pass", token=tok_stu, body={"reason": "x"})
check("学生执行 admin-pass -> 403", code == 403, "code=%d" % code)
# 4c 班委对公示中申请再做 review-pass -> 409 非法流转（见第 5 节 a4 进入公示中后验证）
# 4d 未登录访问受保护接口 -> 401
code, _ = req("GET", "/applications")
check("未登录访问 -> 401", code == 401, "code=%d" % code)
# 4e 跨班异议 -> 403（晏陆君公信23 对 李明 公信2301 的公示申请提异议）
code, _ = req("POST", "/applications/s1/object", token=tok_stu, body={"reason": "造假"})
check("跨班提异议 -> 403", code == 403, "code=%d" % code)

print("\n=== 5. 审核生命周期：公示保护期守卫 ===")
# 班委初审通过 a4（晏陆君）-> 公示中
code, rpass = req("POST", "/applications/%s/review-pass" % app_a4, token=tok_rev, body={"reason": "材料齐全，初审通过"})
check("班委初审通过 a4 -> 200 公示中", code == 200 and (rpass.get("status") == "公示中" if isinstance(rpass, dict) else False),
      (rpass.get("status") if isinstance(rpass, dict) else str(rpass)))
# 刚进入公示，手动结束应被守卫拦下（至少满 1 小时）
code, guard = req("POST", "/applications/%s/publicity-finish" % app_a4, token=tok_rev, body={})
guard_msg = guard.get("error", "") if isinstance(guard, dict) else str(guard)
check("公示<1h 手动结束被守卫拦截 -> 400", code == 400, "code=%d msg=%s" % (code, guard_msg[:40]))
# 4c 班委对同一公示中申请再做 review-pass -> 409 非法流转
code, _ = req("POST", "/applications/%s/review-pass" % app_a4, token=tok_rev, body={})
check("对已公示申请 review-pass -> 409 非法流转", code == 409, "code=%d" % code)

print("\n=== 6. 完整生命周期到「已通过」+ 成绩聚合（用种子 s1，公示窗口 32h 可直接结束）===")
def fnum(d, k):
    v = d.get(k) if isinstance(d, dict) else None
    try:
        return float(v)
    except Exception:
        return None

# 先看李明当前成绩（s1 公示中、s5 已驳回，quality=0）
code, before = req("GET", "/scores/20230001", token=tok_adm)
before_ass = fnum(before, "assessmentScore")
check("李明初始成绩可读（academic 86.4*0.9=77.8）", code == 200 and before_ass is not None and abs(before_ass - 77.8) < 0.01,
      "code=%d assessment=%s" % (code, before_ass))
# 结束公示 -> 待管理员审核（s1 属 公信2301，管理员已授权管辖）
code, pf = req("POST", "/applications/s1/publicity-finish", token=tok_adm, body={})
check("s1 结束公示 -> 待管理员审核", code == 200 and (pf.get("status") == "待管理员审核" if isinstance(pf, dict) else False),
      (pf.get("status") if isinstance(pf, dict) else str(pf)))
# 管理员复审通过（改分需理由）
code, ap = req("POST", "/applications/s1/admin-pass", token=tok_adm, body={"reason": "复审通过，计分入账"})
check("s1 管理员复审通过 -> 已通过", code == 200 and (ap.get("status") == "已通过" if isinstance(ap, dict) else False),
      (ap.get("status") if isinstance(ap, dict) else str(ap)))
# 成绩聚合：academic 86.4*0.9 + quality(12)*0.1 = 77.76+1.2=78.96 -> round=79.0
code, after = req("GET", "/scores/20230001", token=tok_adm)
after_ass = fnum(after, "assessmentScore")
after_q = fnum(after, "qualityScore")
check("s1 已通过后 qualityScore=12", after_q is not None and abs(after_q - 12) < 0.01, "quality=%s" % after_q)
check("成绩聚合 = 79.0（公式 round((86.4*0.9+12*0.1)*10)/10）", after_ass is not None and abs(after_ass - 79.0) < 0.01,
      "assessment=%s" % after_ass)

print("\n=== 7. 申诉路径（已驳回 -> 申诉中 -> 驳回申诉回到已驳回）===")
# 李明对 s5（已驳回）申诉
code, ap1 = req("POST", "/applications/s5/appeal", token=tok_lim, body={"reason": "材料已补齐，申请复核"})
check("对已驳回申请申诉 -> 申诉中", code == 200 and (ap1.get("status") == "申诉中" if isinstance(ap1, dict) else False),
      (ap1.get("status") if isinstance(ap1, dict) else str(ap1)))
# 管理员驳回申诉（s5 claimed==score -> 回到已驳回）
code, ap2 = req("POST", "/applications/s5/appeal-reject", token=tok_adm, body={"reason": "仍缺盖章证明"})
check("驳回申诉 -> 已驳回", code == 200 and (ap2.get("status") == "已驳回" if isinstance(ap2, dict) else False),
      (ap2.get("status") if isinstance(ap2, dict) else str(ap2)))

print("\n=== 8. 公示墙 / 活动列表可读性 ===")
code, pub = req("GET", "/publicity", token=tok_stu)
check("公示墙可读", code == 200 and isinstance(pub, list), "count=%s" % (len(pub) if isinstance(pub, list) else pub))
code, acts = req("GET", "/activities", token=tok_stu)
check("活动列表可读", code == 200 and isinstance(acts, list), "count=%s" % (len(acts) if isinstance(acts, list) else acts))

print("\n================ 结果汇总 ================")
print("PASS: %d   FAIL: %d" % (len(PASS), len(FAIL)))
for f in FAIL:
    print("  FAIL -> " + f)
sys.exit(1 if FAIL else 0)
