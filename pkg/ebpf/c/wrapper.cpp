#include "wrapper.h"

#ifdef __cplusplus
extern "C" {
#endif

#if defined(__GLIBC__) && !defined(SKIP_GLIBC_WRAPPER)

#define wrap_d1_defition(func)                                  \
double __wrap_ ## func (double x) {                             \
  return __ ## func ## _prior_glibc(x);                         \
}

#define wrap_d2_defition(func)                                  \
double __wrap_ ## func (double x, double y) {                   \
  return __ ## func ## _prior_glibc(x, y);                      \
}

#define wrap_f1_defition(func)                                  \
float __wrap_ ## func (float x) {                               \
  return __ ## func ## _prior_glibc(x);                         \
}

#else

// Use functions directly for non-GLIBC environments.

#define wrap_d1_defition(func)                                  \
double __wrap_ ## func (double x) {                             \
  return func(x);                                               \
}

#define wrap_d2_defition(func)                                  \
double __wrap_ ## func (double x, double y) {                   \
  return func(x, y);                                            \
}

#define wrap_f1_defition(func)                                  \
float __wrap_ ## func (float x) {                               \
  return func(x);                                               \
}

#endif

wrap_d1_defition(exp)
wrap_d1_defition(log)
wrap_d2_defition(pow)
wrap_d1_defition(log2)
wrap_f1_defition(log2f)

#ifdef __cplusplus
}
#endif