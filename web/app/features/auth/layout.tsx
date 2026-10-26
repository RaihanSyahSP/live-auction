import { Outlet } from "react-router"

const AuthLayout = () => {
  return (
    <div className="page">
      <Outlet />
    </div>
  )
}

export default AuthLayout
