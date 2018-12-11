/**
 * @file    ast_exp.h
 * @copyright defined in aergo/LICENSE.txt
 */

#ifndef _AST_EXP_H
#define _AST_EXP_H

#include "common.h"

#include "ast.h"
#include "meta.h"
#include "value.h"

#define is_null_exp(exp)            ((exp)->kind == EXP_NULL)
#define is_lit_exp(exp)             ((exp)->kind == EXP_LIT)
#define is_ref_exp(exp)             ((exp)->kind == EXP_REF)
#define is_array_exp(exp)           ((exp)->kind == EXP_ARRAY)
#define is_cast_exp(exp)            ((exp)->kind == EXP_CAST)
#define is_unary_exp(exp)           ((exp)->kind == EXP_UNARY)
#define is_binary_exp(exp)          ((exp)->kind == EXP_BINARY)
#define is_ternary_exp(exp)         ((exp)->kind == EXP_TERNARY)
#define is_access_exp(exp)          ((exp)->kind == EXP_ACCESS)
#define is_call_exp(exp)            ((exp)->kind == EXP_CALL)
#define is_sql_exp(exp)             ((exp)->kind == EXP_SQL)
#define is_tuple_exp(exp)           ((exp)->kind == EXP_TUPLE)
#define is_init_exp(exp)            ((exp)->kind == EXP_INIT)

#define is_usable_lval(exp)                                                              \
    ((exp)->id != NULL && !is_const_id((exp)->id) &&                                     \
     (is_ref_exp(exp) || is_array_exp(exp) || is_access_exp(exp)))

#define exp_add_first               array_add_first
#define exp_add_last                array_add_last

#ifndef _AST_EXP_T
#define _AST_EXP_T
typedef struct ast_exp_s ast_exp_t;
#endif /* ! _AST_EXP_T */

#ifndef _AST_ID_T
#define _AST_ID_T
typedef struct ast_id_s ast_id_t;
#endif /* ! _AST_ID_T */

/* null, true, false, 1, 1.0, 0x1, "..." */
typedef struct exp_lit_s {
    value_t val;
} exp_lit_t;

/* name */
typedef struct exp_ref_s {
    char *name;
} exp_ref_t;

/* id[idx] */
typedef struct exp_array_s {
    ast_exp_t *id_exp;
    ast_exp_t *idx_exp;
} exp_array_t;

/* (type)val */
typedef struct exp_cast_s {
    ast_exp_t *val_exp;
    meta_t to_meta;
} exp_cast_t;

/* id(param, ...) */
typedef struct exp_call_s {
    ast_exp_t *id_exp;
    array_t *param_exps;
} exp_call_t;

/* id.fld */
typedef struct exp_access_s {
    ast_exp_t *id_exp;
    ast_exp_t *fld_exp;
} exp_access_t;

/* val kind */
typedef struct exp_unary_s {
    op_kind_t kind;
    ast_exp_t *val_exp;
} exp_unary_t;

/* l kind r */
typedef struct exp_binary_s {
    op_kind_t kind;
    ast_exp_t *l_exp;
    ast_exp_t *r_exp;
} exp_binary_t;

/* prefix ? infix : postfix */
typedef struct exp_ternary_s {
    ast_exp_t *pre_exp;
    ast_exp_t *in_exp;
    ast_exp_t *post_exp;
} exp_ternary_t;

/* dml, query */
typedef struct exp_sql_s {
    sql_kind_t kind;
    char *sql;
} exp_sql_t;

/* (exp, exp, exp, ...) */
typedef struct exp_tuple_s {
    array_t *exps;
} exp_tuple_t;

/* new {exp, exp, exp, ...} */
typedef struct exp_init_s {
    array_t *exps;
} exp_init_t;

struct ast_exp_s {
    AST_NODE_DECL;

    exp_kind_t kind;

    union {
        exp_lit_t u_lit;
        exp_ref_t u_ref;
        exp_array_t u_arr;
        exp_cast_t u_cast;
        exp_call_t u_call;
        exp_access_t u_acc;
        exp_unary_t u_un;
        exp_binary_t u_bin;
        exp_ternary_t u_tern;
        exp_sql_t u_sql;
        exp_tuple_t u_tup;
        exp_init_t u_init;
    };

    /* results of semantic checker */
    ast_id_t *id;       /* referenced identifier */
    meta_t meta;
};

ast_exp_t *exp_new_null(src_pos_t *pos);
ast_exp_t *exp_new_lit(src_pos_t *pos);
ast_exp_t *exp_new_ref(char *name, src_pos_t *pos);
ast_exp_t *exp_new_array(ast_exp_t *id_exp, ast_exp_t *idx_exp, src_pos_t *pos);
ast_exp_t *exp_new_cast(type_t type, ast_exp_t *val_exp, src_pos_t *pos);
ast_exp_t *exp_new_call(ast_exp_t *id_exp, array_t *param_exps, src_pos_t *pos);
ast_exp_t *exp_new_access(ast_exp_t *id_exp, ast_exp_t *fld_exp, src_pos_t *pos);
ast_exp_t *exp_new_unary(op_kind_t kind, ast_exp_t *val_exp, src_pos_t *pos);
ast_exp_t *exp_new_binary(op_kind_t kind, ast_exp_t *l_exp, ast_exp_t *r_exp,
                          src_pos_t *pos);
ast_exp_t *exp_new_ternary(ast_exp_t *pre_exp, ast_exp_t *in_exp, ast_exp_t *post_exp,
                           src_pos_t *pos);
ast_exp_t *exp_new_sql(sql_kind_t kind, char *sql, src_pos_t *pos);
ast_exp_t *exp_new_tuple(array_t *exps, src_pos_t *pos);
ast_exp_t *exp_new_init(array_t *exps, src_pos_t *pos);

ast_exp_t *exp_clone(ast_exp_t *exp);

void ast_exp_dump(ast_exp_t *exp, int indent);

#endif /* ! _AST_EXP_H */