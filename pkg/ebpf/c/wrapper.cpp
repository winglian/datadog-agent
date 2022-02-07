#include "wrapper.h"

#ifdef __cplusplus
extern "C" {
#endif

#if defined(__GLIBC__) && !defined(SKIP_GLIBC_WRAPPER)

#define symver_wrap_d1(func)                                    \
double __wrap_ ## func (double x) {                             \
  return __ ## func ## _prior_glibc(x);                         \
}

#define symver_wrap_d2(func)                                    \
double __wrap_ ## func (double x, double y) {                   \
  return __ ## func ## _prior_glibc(x, y);                      \
}

#define symver_wrap_f1(func)                                    \
float __wrap_ ## func (float x) {                               \
  return __ ## func ## _prior_glibc(x);                         \
}

#else

// Use functions directly for non-GLIBC environments.

#define symver_wrap_d1(func)                                    \
double __wrap_ ## func (double x) {                             \
  return func(x);                                               \
}

#define symver_wrap_d2(func)                                    \
double __wrap_ ## func (double x, double y) {                   \
  return func(x, y);                                            \
}

#define symver_wrap_f1(func)                                    \
float __wrap_ ## func (float x) {                               \
  return func(x);                                               \
}

#endif

symver_wrap_d1(exp)
symver_wrap_d1(log)
symver_wrap_d2(pow)
symver_wrap_d1(log2)
symver_wrap_f1(log2f)

#ifdef __cplusplus
}
#endif