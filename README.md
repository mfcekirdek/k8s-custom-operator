# k8s-custom-operator
A PoC repo for creating custom k8s operator

#references 
https://book.kubebuilder.io/quick-start.html#test-it-out

# To create project
kubebuilder init --domain order.io --repo github.com/order  

# To create api
kubebuilder create api --group webapp --version v1 --kind Order 

# To create CRD
make manifests; make generate; make install;